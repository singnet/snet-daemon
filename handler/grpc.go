package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/bufbuild/protocompile/linker"
	"github.com/gorilla/rpc/v2/json2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
)

var grpcDesc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

type grpcHandler struct {
	grpcConn            *grpc.ClientConn
	grpcModelConn       *grpc.ClientConn
	options             grpc.DialOption
	enc                 string
	passthroughEndpoint string
	//modelTrainingEndpoint string
	executable         string
	serviceMetaData    *blockchain.ServiceMetadata
	serviceCredentials serviceCredentials
}

func (g grpcHandler) GrpcConn(isModelTraining bool) *grpc.ClientConn {
	if isModelTraining {
		return g.grpcModelConn
	}

	return g.grpcConn
}

func NewGrpcHandler(serviceMetadata *blockchain.ServiceMetadata) grpc.StreamHandler {
	passthroughEnabled := config.GetBool(config.PassthroughEnabledKey)

	if !passthroughEnabled {
		return grpcLoopback
	}

	h := grpcHandler{
		serviceMetaData:     serviceMetadata,
		enc:                 serviceMetadata.GetWireEncoding(),
		passthroughEndpoint: config.GetString(config.ServiceEndpointKey),
		//modelTrainingEndpoint: config.GetString(config.ModelTrainingEndpoint),
		executable: config.GetString(config.ExecutablePathKey),
		options: grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
			grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024)),
	}

	switch serviceMetadata.GetServiceType() {
	case "grpc":
		h.grpcConn = h.getConnection(h.passthroughEndpoint)
		//if config.GetBool(config.ModelTrainingEnabled) {
		//	h.grpcModelConn = h.getConnection(h.modelTrainingEndpoint)
		//}
		return h.grpcToGRPC
	case "jsonrpc":
		return h.grpcToJSONRPC
	case "http":
		h.serviceCredentials = serviceCredentials{}
		err := config.Vip().UnmarshalKey(config.ServiceCredentialsKey, &h.serviceCredentials)
		if err != nil {
			zap.L().Fatal("invalid config", zap.Error(fmt.Errorf("%v%v", err, errs.ErrDescURL(errs.InvalidServiceCredentials))))
		}
		err = h.serviceCredentials.validate()
		if err != nil {
			zap.L().Fatal("invalid config", zap.Error(fmt.Errorf("%v%v", err, errs.ErrDescURL(errs.InvalidServiceCredentials))))
		}
		return h.grpcToHTTP
	case "process":
		return h.grpcToProcess
	}
	return nil
}

func (srvCreds serviceCredentials) validate() error {
	if len(srvCreds) > 0 {
		for _, v := range srvCreds {
			if v.Location != body && v.Location != header && v.Location != query {
				return fmt.Errorf("invalid service_credentials: location should be body, header or query")
			}
			if v.Key == "" {
				return fmt.Errorf("invalid service_credentials: key can't be empty")
			}
		}
	}
	return nil
}

func (g grpcHandler) getConnection(endpoint string) (conn *grpc.ClientConn) {

	if !strings.Contains(endpoint, "://") {
		endpoint = "grpc" + "://" + endpoint
	}

	passthroughURL, err := url.Parse(endpoint)
	if err != nil || passthroughURL == nil {
		zap.L().Fatal(fmt.Sprintf("can't parse service_endpoint %v", errs.ErrDescURL(errs.InvalidConfig)), zap.String("endpoint", endpoint))
	}
	if strings.Compare(passthroughURL.Scheme, "https") == 0 {
		conn, err = grpc.NewClient(passthroughURL.Host,
			grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), g.options)
		if err != nil {
			zap.L().Panic("error dialing service", zap.Error(err))
		}
		return conn
	}
	conn, err = grpc.NewClient(passthroughURL.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), g.options)
	if err != nil {
		zap.L().Panic("error dialing service", zap.Error(err))
	}
	return conn
}

/*
Modified from https://github.com/mwitkow/grpc-proxy/blob/67591eb23c48346a480470e462289835d96f70da/proxy/handler.go#L61
Original Copyright 2017 Michal Witkowski. All Rights Reserved. See LICENSE-GRPC-PROXY for licensing terms.
Modifications Copyright 2018 SingularityNET Foundation. All Rights Reserved. See LICENSE for licensing terms.
*/
func (g grpcHandler) grpcToGRPC(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)
	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	inCtx := inStream.Context()
	md, ok := metadata.FromIncomingContext(inCtx)
	if !ok {
		return status.Errorf(codes.Internal, "could not get metadata from incoming context")
	}

	outCtx, outCancel := context.WithCancel(inCtx)
	defer outCancel()
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())
	isModelTraining := g.serviceMetaData.IsModelTraining(method)
	outStream, err := g.GrpcConn(isModelTraining).NewStream(outCtx, grpcDesc, method, grpc.CallContentSubtype(g.enc))
	if err != nil {
		return status.Errorf(codes.Internal, "can't connect to service %v%v", err, errs.ErrDescURL(errs.ServiceUnavailable))
	}

	s2cErrChan := forwardServerToClient(inStream, outStream)
	c2sErrChan := forwardClientToServer(outStream, inStream)

	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>inStream may continue pumping though.
				errCloseSend := outStream.CloseSend()
				if errCloseSend != nil {
					zap.L().Debug("failed close outStream", zap.Error(err))
				}
				break
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				outCancel()
				return status.Errorf(codes.Internal, "failed proxying s2c: %v%s", s2cErr, errs.ErrDescURL(errs.ServiceUnavailable))
			}
		case c2sErr := <-c2sErrChan:
			// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two
			// cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers
			// will be nil.
			inStream.SetTrailer(outStream.Trailer())
			// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
			if c2sErr != io.EOF {
				return c2sErr
			}
			return nil
		}
	}
	return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

/*
Modified from https://github.com/mwitkow/grpc-proxy/blob/67591eb23c48346a480470e462289835d96f70da/proxy/handler.go#L115
Original Copyright 2017 Michal Witkowski. All Rights Reserved. See LICENSE-GRPC-PROXY for licensing terms.
Modifications Copyright 2018 SingularityNET Foundation. All Rights Reserved. See LICENSE for licensing terms.
*/
func forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &codec.GrpcFrame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if i == 0 {
				// This is a bit of a hack, but client to server headers are only readable after first client msg is
				// received but must be written to server stream before the first msg is flushed.
				// This is the only place to do it nicely.
				md, err := src.Header()
				if err != nil {
					ret <- err
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

/*
Modified from https://github.com/mwitkow/grpc-proxy/blob/67591eb23c48346a480470e462289835d96f70da/proxy/handler.go#L147
Original Copyright 2017 Michal Witkowski. All Rights Reserved. See LICENSE-GRPC-PROXY for licensing terms.
Modifications Copyright 2018 SingularityNET Foundation. All Rights Reserved. See LICENSE for licensing terms.
*/
func forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &codec.GrpcFrame{}
		for i := 0; ; i++ {
			//Only for the first time do this, once RecvMsg has been called,
			//future calls will result in io.EOF , we want to retrieve the
			// first message sent by the client and pass this on the regular service call
			//This is done to be able to make calls to support regular Service call + Dynamic pricing call
			if i == 0 {
				// todo we need to think through to determine price for every call on stream calls
				// will be handled when we support streaming and pricing across all clients in snet-platform
				if wrappedStream, ok := src.(*WrapperServerStream); ok {
					f = (wrappedStream.OriginalRecvMsg()).(*codec.GrpcFrame)
				} else if err := src.RecvMsg(f); err != nil {
					ret <- err
					break
				}
			} else if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

type httpLocation string

var query httpLocation = "query"
var body httpLocation = "body"
var header httpLocation = "header"

type serviceCredential struct {
	Key      string       `json:"key"`
	Value    any          `json:"value"`
	Location httpLocation `json:"location"`
}

type serviceCredentials []serviceCredential

func (g grpcHandler) grpcToHTTP(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)
	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	methodSegs := strings.Split(method, "/")
	method = methodSegs[len(methodSegs)-1]

	zap.L().Info("Calling method", zap.String("method", method))

	// Check if this is a WrapperServerStream with pre-read data
	var f *codec.GrpcFrame
	if wrapper, ok := inStream.(*WrapperServerStream); ok {
		if recvMsg := wrapper.OriginalRecvMsg(); recvMsg != nil {
			if frame, ok := recvMsg.(*codec.GrpcFrame); ok {
				f = frame
				zap.L().Debug("Found pre-read GrpcFrame", zap.Int("dataLen", len(f.Data)), zap.String("data", string(f.Data)))
			} else {
				return status.Errorf(codes.Internal, "pre-read message is not GrpcFrame %T", recvMsg)
			}
		} else {
			return status.Errorf(codes.Internal, "no pre-read message in WrapperServerStream")
		}
	} else {
		// Fallback to reading from stream (for non-wrapper streams)
		zap.L().Debug("Not a WrapperServerStream, reading from stream directly")
		f = &codec.GrpcFrame{}
		if err := inStream.RecvMsg(f); err != nil {
			return status.Errorf(codes.Internal, "error receiving grpc msg: %v%v", err, errs.ErrDescURL(errs.ReceiveMsgError))
		}
	}

	// convert proto msg to json
	jsonBody, err := protoToJson(g.serviceMetaData.ProtoDescriptors, f.Data, method)
	if err != nil {
		return status.Errorf(codes.Internal, "protoToJson error: %+v", errs.ErrDescURL(errs.InvalidProto))
	}

	zap.L().Debug("Proto to json result", zap.String("json", string(jsonBody)))

	base, err := url.Parse(g.passthroughEndpoint)
	if err != nil {
		zap.L().Error("can't parse passthroughEndpoint", zap.Error(err))
		return status.Errorf(codes.Internal, "can't parse service_endpoint %v%v", err, errs.ErrDescURL(errs.InvalidConfig))
	}

	base.Path += method // method from proto should be the same as http handler path

	params := url.Values{}
	headers := http.Header{}

	var bodyMap = map[string]any{}
	errJson := json.Unmarshal(jsonBody, &bodyMap)

	for _, cred := range g.serviceCredentials {
		switch cred.Location {
		case query:
			v, ok := cred.Value.(string)
			if ok {
				params.Add(cred.Key, v)
			}
		case body:
			if errJson == nil {
				bodyMap[cred.Key] = cred.Value
			}
		case header:
			v, ok := cred.Value.(string)
			if ok {
				headers.Set(cred.Key, v)
			}
		}
	}

	if errJson == nil {
		newJson, err := json.Marshal(bodyMap)
		if err == nil {
			jsonBody = newJson
		} else {
			zap.L().Debug("Can't marshal json", zap.Error(err))
		}
	}

	base.RawQuery = params.Encode()
	zap.L().Debug("Calling http service",
		zap.String("url", base.String()),
		zap.String("body", string(jsonBody)),
		zap.String("method", "POST"))

	httpReq, err := http.NewRequest("POST", base.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return status.Errorf(codes.Internal, "error creating http request: %+v%v", err, errs.ErrDescURL(errs.HTTPRequestBuildError))
	}
	httpReq.Header = headers
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return status.Errorf(codes.Internal, "error executing HTTP service: %+v%v", err, errs.ErrDescURL(errs.ServiceUnavailable))
	}
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return status.Errorf(codes.Internal, "error reading response from HTTP service: %+v%v", err, errs.ErrDescURL(errs.ServiceUnavailable))
	}
	zap.L().Debug("Response from HTTP service", zap.String("response", string(resp)))

	protoMessage, errMarshal := jsonToProto(g.serviceMetaData.ProtoDescriptors, resp, method)
	if errMarshal != nil {
		return status.Errorf(codes.Internal, "jsonToProto error: %+v%v", errMarshal, errs.ErrDescURL(errs.InvalidProto))
	}

	if err = inStream.SendMsg(protoMessage); err != nil {
		return status.Errorf(codes.Internal, "error sending response from HTTP service: %+v", err)
	}

	return nil
}

func findMethodInProto(protoFiles linker.Files, methodName string) (method protoreflect.MethodDescriptor) {
	for _, protoFile := range protoFiles {
		if protoFile.Services().Len() == 0 {
			continue
		}

		for i := 0; i < protoFile.Services().Len(); i++ {
			service := protoFile.Services().Get(i)
			if service == nil {
				continue
			}

			method = service.Methods().ByName(protoreflect.Name(methodName))
			if method != nil {
				return method
			}
		}
	}
	return nil
}

func jsonToProto(protoFiles linker.Files, json []byte, methodName string) (proto proto.Message, err error) {

	method := findMethodInProto(protoFiles, methodName)
	if method == nil {
		zap.L().Error("[jsonToProto] method not found in proto for http call")
		return proto, errors.New("method in proto not found")
	}

	output := method.Output()
	zap.L().Debug("output msg descriptor", zap.String("fullname", string(output.FullName())))
	proto = dynamicpb.NewMessage(output)
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(json, proto)
	if err != nil {
		zap.L().Error("can't unmarshal json to proto", zap.Error(err))
		return proto, fmt.Errorf("invalid proto, can't convert json to proto msg: %+v", err)
	}

	return proto, nil
}

func protoToJson(protoFiles linker.Files, in []byte, methodName string) (json []byte, err error) {

	method := findMethodInProto(protoFiles, methodName)
	if method == nil {
		zap.L().Error("[protoToJson] method not found in proto for http call")
		return []byte("error, method in proto not found"), errors.New("method in proto not found")
	}

	input := method.Input()
	zap.L().Debug("[protoToJson]", zap.Any("methodName", input.FullName()))
	msg := dynamicpb.NewMessage(input)
	err = proto.Unmarshal(in, msg)
	if err != nil {
		zap.L().Error("Error in unmarshalling", zap.Error(err))
		return []byte("error, invalid proto file or input request"), fmt.Errorf("error in unmarshaling proto to json: %+v", err)
	}
	json, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		zap.L().Error("Error in marshaling", zap.Error(err))
		return []byte("error, invalid proto file or input request"), fmt.Errorf("error in marshaling proto to json: %+v", err)
	}
	zap.L().Debug("ProtoToJson result:", zap.String("json", string(json)))

	return json, nil
}

func (g grpcHandler) grpcToJSONRPC(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)
	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	methodSegs := strings.Split(method, "/")
	method = methodSegs[len(methodSegs)-1]

	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error receiving request; error: %+v", err)
	}

	params := new(any)

	if err := json.Unmarshal(f.Data, params); err != nil {
		return status.Errorf(codes.Internal, "error unmarshaling request; error: %+v", err)
	}

	jsonRPCReq, err := json2.EncodeClientRequest(method, params)

	if err != nil {
		return status.Errorf(codes.Internal, "error encoding request; error: %+v", err)
	}

	httpReq, err := http.NewRequest("POST", g.passthroughEndpoint, bytes.NewBuffer(jsonRPCReq))
	if err != nil {
		return status.Errorf(codes.Internal, "error creating http request; error: %+v", err)
	}

	httpReq.Header.Set("content-type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)

	if err != nil {
		return status.Errorf(codes.Internal, "error executing http call; error: %+v", err)
	}

	result := new(any)

	if err = json2.DecodeClientResponse(httpResp.Body, result); err != nil {
		return status.Errorf(codes.Internal, "json-rpc error; error: %+v", err)
	}

	respBytes, err := json.Marshal(result)

	if err != nil {
		return status.Errorf(codes.Internal, "error marshaling response; error: %+v", err)
	}

	f = &codec.GrpcFrame{Data: respBytes}

	if err = inStream.SendMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+v", err)
	}

	return nil
}

type WrapperServerStream struct {
	sendHeaderCalled bool
	stream           grpc.ServerStream
	recvMessage      any
	sentMessage      any
	Ctx              context.Context
}

func (f *WrapperServerStream) SetTrailer(md metadata.MD) {
	f.stream.SetTrailer(md)
}

func NewWrapperServerStream(stream grpc.ServerStream, ctx context.Context) (grpc.ServerStream, error) {
	m := &codec.GrpcFrame{}
	err := stream.RecvMsg(m)
	f := &WrapperServerStream{
		stream:           stream,
		recvMessage:      m,
		sendHeaderCalled: false,
		Ctx:              ctx, // save modified ctx
	}
	return f, err
}

func (f *WrapperServerStream) Context() context.Context {
	// old way return f.stream.Context()
	return f.Ctx // return modified context
}

func (f *WrapperServerStream) SetHeader(md metadata.MD) error {
	return f.stream.SetHeader(md)
}

func (f *WrapperServerStream) SendHeader(md metadata.MD) error {
	//this is more of a hack to support dynamic pricing
	// when the service method returns the price in cogs, the SendHeader will be called,
	// we don't want this as the SendHeader can be called just once in the ServerStream
	if !f.sendHeaderCalled {
		return nil
	}
	f.sendHeaderCalled = true
	return f.stream.SendHeader(md)
}

func (f *WrapperServerStream) SendMsg(m any) error {
	return f.stream.SendMsg(m)
}

func (f *WrapperServerStream) RecvMsg(m any) error {
	return f.stream.RecvMsg(m)
}

func (f *WrapperServerStream) OriginalRecvMsg() any {
	return f.recvMessage
}

func (g grpcHandler) grpcToProcess(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)

	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	methodSegs := strings.Split(method, "/")
	method = methodSegs[len(methodSegs)-1]

	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error receiving request; error: %+v", err)
	}

	cmd := exec.Command(g.executable, method)
	stdin, err := cmd.StdinPipe()

	if err != nil {
		return status.Errorf(codes.Internal, "error creating stdin pipe; error: %+v", err)
	}

	if _, err := stdin.Write(f.Data); err != nil {
		return status.Errorf(codes.Internal, "error writing to stdin; error: %+v", err)
	}
	stdin.Close()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return status.Errorf(codes.Internal, "error executing process; error: %+v", err)
	}

	f = &codec.GrpcFrame{Data: out}

	if err = inStream.SendMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+v", err)
	}

	return nil
}

func grpcLoopback(srv any, inStream grpc.ServerStream) error {
	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error receiving request; error: %+v", err)
	}

	if err := inStream.SendMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+v", err)
	}

	return nil
}
