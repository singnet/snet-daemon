package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/codec"
	"github.com/singnet/snet-daemon/config"

	"github.com/gorilla/rpc/v2/json2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var grpcDesc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

type grpcHandler struct {
	grpcConn              *grpc.ClientConn
	grpcModelConn         *grpc.ClientConn
	options               grpc.DialOption
	enc                   string
	passthroughEndpoint   string
	modelTrainingEndpoint string
	executable            string
	serviceMetaData       *blockchain.ServiceMetadata
	serviceCredentials    []serviceCredential
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
		serviceMetaData:       serviceMetadata,
		enc:                   serviceMetadata.GetWireEncoding(),
		passthroughEndpoint:   config.GetString(config.PassthroughEndpointKey),
		modelTrainingEndpoint: config.GetString(config.ModelTrainingEndpoint),
		executable:            config.GetString(config.ExecutablePathKey),
		options: grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
			grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024)),
	}

	switch serviceMetadata.GetServiceType() {
	case "grpc":
		h.grpcConn = h.getConnection(h.passthroughEndpoint)
		if config.GetBool(config.ModelTrainingEnabled) {
			h.grpcModelConn = h.getConnection(h.modelTrainingEndpoint)
		}
		return h.grpcToGRPC
	case "jsonrpc":
		return h.grpcToJSONRPC
	case "http":
		h.serviceCredentials = []serviceCredential{}
		err := config.Vip().UnmarshalKey(config.ServiceCredentialsKey, &h.serviceCredentials)
		if err != nil {
			zap.L().Panic("invalid config", zap.Error(err))
		}
		return h.grpcToHTTP
	case "process":
		return h.grpcToProcess
	}
	return nil
}

func (h grpcHandler) getConnection(endpoint string) (conn *grpc.ClientConn) {
	passthroughURL, err := url.Parse(endpoint)
	if err != nil {
		zap.L().Panic("error parsing passthrough endpoint", zap.Error(err))
	}
	if strings.Compare(passthroughURL.Scheme, "https") == 0 {
		conn, err = grpc.Dial(passthroughURL.Host,
			grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), h.options)
		if err != nil {
			zap.L().Panic("error dialing service", zap.Error(err))
		}
	} else {
		conn, err = grpc.Dial(passthroughURL.Host, grpc.WithInsecure(), h.options)

		if err != nil {
			zap.L().Panic("error dialing service", zap.Error(err))
		}
	}
	return
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
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())
	isModelTraining := g.serviceMetaData.IsModelTraining(method)
	outStream, err := g.GrpcConn(isModelTraining).NewStream(outCtx, grpcDesc, method, grpc.CallContentSubtype(g.enc))
	if err != nil {
		return err
	}

	s2cErrChan := forwardServerToClient(inStream, outStream)
	c2sErrChan := forwardClientToServer(outStream, inStream)

	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>inStream may continue pumping though.
				outStream.CloseSend()
				break
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				outCancel()
				return status.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
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
				//todo we need to think through to determine price for every call on stream calls
				//will be handled when we support streaming and pricing across all clients in snet-platform
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

func (g grpcHandler) grpcToHTTP(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)

	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	methodSegs := strings.Split(method, "/")
	method = methodSegs[len(methodSegs)-1]

	zap.L().Info("Calling method", zap.String("method", method))

	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		zap.L().Error(err.Error())
		return status.Errorf(codes.Internal, "error receiving request; error: %+cred", err)
	}

	// convert proto msg to json
	jsonBody := protoToJson(g.serviceMetaData.ProtoFile, f.Data, method)

	zap.L().Debug("Proto to json", zap.String("json", string(jsonBody)))

	base, err := url.Parse(g.passthroughEndpoint)
	if err != nil {
		zap.L().Error("can't parse passthroughEndpoint", zap.Error(err))
	}

	base.Path += method // method from proto should be the same as http handler path

	params := url.Values{}
	headers := http.Header{}

	var bodymap = map[string]any{}
	errJson := json.Unmarshal(jsonBody, &bodymap)

	for _, cred := range g.serviceCredentials {
		switch cred.Location {
		case query:
			v, ok := cred.Value.(string)
			if ok {
				params.Add(cred.Key, v)
			}
		case body:
			if errJson == nil {
				bodymap[cred.Key] = cred.Value
			}
		case header:
			v, ok := cred.Value.(string)
			if ok {
				headers.Set(cred.Key, v)
			}
		}
	}

	if errJson == nil {
		newJson, err := json.Marshal(bodymap)
		if err == nil {
			jsonBody = newJson
		} else {
			zap.L().Debug("Can't marshal json", zap.Error(err))
		}
	}

	base.RawQuery = params.Encode()
	zap.L().Debug("Calling URL", zap.String("Url", base.String()))
	httpReq, err := http.NewRequest("POST", base.String(), bytes.NewBuffer(jsonBody))
	httpReq.Header = headers
	if err != nil {
		return status.Errorf(codes.Internal, "error creating http request; error: %+cred", err)
	}

	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)

	if err != nil {
		return status.Errorf(codes.Internal, "error executing http call; error: %+cred", err)
	}

	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return status.Errorf(codes.Internal, "error reading response; error: %+cred", err)
	}
	zap.L().Debug("Getting response", zap.String("response", string(resp)))

	protoMessage := jsonToProto(g.serviceMetaData.ProtoFile, resp, method)
	if err = inStream.SendMsg(protoMessage); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+cred", err)
	}

	return nil
}

func jsonToProto(protoFile protoreflect.FileDescriptor, json []byte, methodName string) (proto proto.Message) {

	zap.L().Debug("Processing file", zap.String("fileName", string(protoFile.Name())))
	zap.L().Debug("Count services: ", zap.Int("value", protoFile.Services().Len()))

	if protoFile.Services().Len() == 0 {
		zap.L().Warn("service in proto not found")
		return proto
	}

	service := protoFile.Services().Get(0)
	if service == nil {
		zap.L().Warn("service in proto not found")
		return proto
	}

	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		zap.L().Warn("method not found in proto")
		return proto
	}
	output := method.Output()
	zap.L().Debug("output of calling method", zap.Any("method", output.FullName()))
	proto = dynamicpb.NewMessage(output)
	err := protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(json, proto)
	if err != nil {
		zap.L().Error("Can't unmarshal jsonToProto", zap.Error(err))
	}

	return proto
}

func protoToJson(protoFile protoreflect.FileDescriptor, in []byte, methodName string) (json []byte) {

	if protoFile.Services().Len() == 0 {
		zap.L().Warn("service in proto not found")
		return []byte("error, invalid proto file")
	}

	service := protoFile.Services().Get(0)
	if service == nil {
		zap.L().Warn("service in proto not found")
		return []byte("error, invalid proto file")
	}

	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		zap.L().Warn("method not found in proto")
		return []byte("error, invalid proto file or input request")
	}

	input := method.Input()
	zap.L().Debug("Input fullname method", zap.Any("value", input.FullName()))
	msg := dynamicpb.NewMessage(input)
	err := proto.Unmarshal(in, msg)
	if err != nil {
		zap.L().Error("Error in unmarshalling", zap.Error(err))
		return []byte("error, invalid proto file or input request")
	}
	json, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		zap.L().Error("Error in marshaling", zap.Error(err))
		return []byte("error, invalid proto file or input request")
	}
	zap.L().Debug("Getting json", zap.String("json", string(json)))

	return json
}

func (g grpcHandler) grpcToJSONRPC(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)

	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	methodSegs := strings.Split(method, "/")
	method = methodSegs[len(methodSegs)-1]

	if !ok {
		return status.Errorf(codes.Internal, "could not get metadata from incoming context")
	}

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
}

func (f *WrapperServerStream) SetTrailer(md metadata.MD) {
	f.stream.SetTrailer(md)
}

func NewWrapperServerStream(stream grpc.ServerStream) (grpc.ServerStream, error) {
	m := &codec.GrpcFrame{}
	err := stream.RecvMsg(m)
	f := &WrapperServerStream{
		stream:           stream,
		recvMessage:      m,
		sendHeaderCalled: false,
	}
	return f, err
}

func (f *WrapperServerStream) SetHeader(md metadata.MD) error {
	return f.stream.SetHeader(md)
}
func (f *WrapperServerStream) SendHeader(md metadata.MD) error {
	//this is more of a hack to support dynamic pricing
	// when the service method returns the price in cogs, the SendHeader, will be called,
	// we dont want this as the SendHeader can be called just once in the ServerStream
	if !f.sendHeaderCalled {
		return nil
	}
	f.sendHeaderCalled = true
	return f.stream.SendHeader(md)

}

func (f *WrapperServerStream) Context() context.Context {
	return f.stream.Context()
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
