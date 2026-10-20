package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/gorilla/rpc/v2/json2"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var grpcDesc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

type grpcHandler struct {
	grpcConn            *grpc.ClientConn
	grpcModelConn       *grpc.ClientConn
	options             grpc.DialOption
	enc                 string
	passthroughEndpoint string
	executable          string
	serviceMetaData     *blockchain.ServiceMetadata
	serviceCredentials  serviceCredentials
	httpClient          *http.Client
	timeout             time.Duration
}

func (g *grpcHandler) GrpcConn(isModelTraining bool) *grpc.ClientConn {
	if isModelTraining {
		return g.grpcModelConn
	}

	return g.grpcConn
}

// NewGrpcHandler selects a backend bridge from service metadata: gRPC, JSON-RPC,
// protobuf-to-HTTP/JSON, or a local process. Disabling passthrough echoes one frame
// instead; payment validation remains the responsibility of the server's interceptors.
func NewGrpcHandler(serviceMetadata *blockchain.ServiceMetadata) grpc.StreamHandler {
	passthroughEnabled := config.GetBool(config.PassthroughEnabledKey)

	if !passthroughEnabled {
		return grpcLoopback
	}

	timeout := config.GetDuration(config.ServiceTimeout)

	h := &grpcHandler{
		timeout:             timeout,
		serviceMetaData:     serviceMetadata,
		enc:                 serviceMetadata.GetWireEncoding(),
		passthroughEndpoint: config.GetString(config.ServiceEndpointKey),
		executable:          config.GetString(config.ExecutablePathKey),
		options: grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
			grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024)),
	}

	// The HTTP client has a separate limit of service_timeout plus one second.
	// An existing caller deadline is preserved, so it may be longer than this limit.
	h.httpClient = &http.Client{Timeout: timeout + time.Second}

	switch serviceMetadata.GetServiceType() {
	case "grpc":
		h.grpcConn = h.getConnection(h.passthroughEndpoint)
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

// getConnection uses TLS only for an https endpoint; other schemes use plaintext gRPC.
func (g *grpcHandler) getConnection(endpoint string) (conn *grpc.ClientConn) {

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
func (g *grpcHandler) grpcToGRPC(srv any, inStream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(inStream)
	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	inCtx := inStream.Context()
	md, ok := metadata.FromIncomingContext(inCtx)
	if !ok {
		return status.Errorf(codes.Internal, "could not get metadata from incoming context")
	}

	outCtx, outCancel := withDefaultTimeout(inCtx, g.timeout)
	defer outCancel()
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())

	isModelTraining := g.serviceMetaData.IsModelTraining(method)
	outStream, err := g.GrpcConn(isModelTraining).NewStream(outCtx, grpcDesc, method, grpc.CallContentSubtype(g.enc))
	if err != nil {
		return status.Errorf(codes.Internal, "can't connect to service %v%v", err, errs.ErrDescURL(errs.ServiceUnavailable))
	}

	s2cErrChan := forwardServerToClient(inStream, outStream)
	c2sErrChan := forwardClientToServer(outStream, inStream)

	for range 2 {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// The caller has finished sending requests. Half-close the backend
				// request stream while continuing to receive backend responses.
				errCloseSend := outStream.CloseSend()
				if errCloseSend != nil {
					zap.L().Debug("failed close outStream", zap.Error(err))
				}
				break
			} else {
				// A frontend receive/forwarding error aborts the backend call.
				outCancel()
				return status.Errorf(codes.Internal, "failed proxying s2c: %v%s", s2cErr, errs.ErrDescURL(errs.ServiceUnavailable))
			}
		case c2sErr := <-c2sErrChan:
			// The backend response stream ended or forwarding failed. Relay any
			// available trailers before returning the final status to the caller.
			inStream.SetTrailer(outStream.Trailer())
			// Preserve backend RPC errors; io.EOF is normal completion.
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
	// The client stream is the daemon's backend connection; the server stream
	// is its frontend connection. Copy backend responses to the original caller.
	ret := make(chan error, 1)
	go func() {
		f := &codec.GrpcFrame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // io.EOF indicates normal backend completion.
				break
			}
			if i == 0 {
				// After receiving the first backend response, forward its headers
				// before sending the response frame to the original caller.
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
	// Copy frontend requests to the daemon's backend client stream.
	ret := make(chan error, 1)
	go func() {
		f := &codec.GrpcFrame{}
		for {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // io.EOF means the caller has finished sending requests.
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

// grpcToHTTP converts one protobuf request to an HTTP JSON POST and converts the
// JSON response back to protobuf. It is a single-request bridge, not a streaming transport.
func (g *grpcHandler) grpcToHTTP(srv any, inStream grpc.ServerStream) error {

	methodFull, ok := grpc.MethodFromServerStream(inStream)
	if !ok {
		return status.Errorf(codes.Internal, "could not determine method from server stream")
	}

	// Normalize surrounding slashes before validating the service/method path.
	methodFull = strings.Trim(methodFull, "/")

	// Require exactly two nonempty path components.
	svc, method, ok := strings.Cut(methodFull, "/")
	if !ok || svc == "" || method == "" {
		return status.Errorf(codes.Internal, "unexpected grpc method format: %q", methodFull)
	}

	if strings.Contains(method, "/") {
		return status.Errorf(codes.Internal, "unexpected grpc method format (extra segments): %q", methodFull)
	}

	zap.L().Info("Calling method", zap.String("method", method))

	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error receiving grpc msg: %v%v", err, errs.ErrDescURL(errs.ReceiveMsgError))
	}

	// Resolve the request type from the compiled proto descriptors.
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

	base.Path = path.Join(base.Path, method) // Append the RPC method name, not the service name.

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

	inCtx := inStream.Context()
	outCtx, cancel := withDefaultTimeout(inCtx, g.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(outCtx, http.MethodPost, base.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return status.Errorf(codes.Internal, "error creating http request: %+v%v", err, errs.ErrDescURL(errs.HTTPRequestBuildError))
	}
	httpReq.Header = headers
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return status.Errorf(codes.Internal, "error executing HTTP service: %+v%v", err, errs.ErrDescURL(errs.ServiceUnavailable))
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		return status.Errorf(codes.Unavailable, "upstream http status %d: %s%v",
			httpResp.StatusCode, string(b), errs.ErrDescURL(errs.ServiceUnavailable))
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

// grpcToJSONRPC treats one incoming frame as JSON parameters for the method's
// final path component and returns the JSON-RPC result as a JSON frame.
func (g *grpcHandler) grpcToJSONRPC(srv any, inStream grpc.ServerStream) error {
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

	inCtx := inStream.Context()
	outCtx, cancel := withDefaultTimeout(inCtx, g.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(outCtx, http.MethodPost, g.passthroughEndpoint, bytes.NewBuffer(jsonRPCReq))
	if err != nil {
		return status.Errorf(codes.Internal, "error creating http request; error: %+v", err)
	}

	httpReq.Header.Set("content-type", "application/json")
	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return status.Errorf(codes.Internal, "error executing http call; error: %+v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		return status.Errorf(codes.Unavailable, "upstream http status %d: %s%v",
			httpResp.StatusCode, string(b), errs.ErrDescURL(errs.ServiceUnavailable))
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

// grpcToProcess passes the method's final path component as an executable argument
// and one request frame on stdin. Combined stdout/stderr becomes the response frame.
func (g *grpcHandler) grpcToProcess(srv any, inStream grpc.ServerStream) error {
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

	inCtx := inStream.Context()
	outCtx, cancel := withDefaultTimeout(inCtx, g.timeout)
	defer cancel()

	cmd := exec.CommandContext(outCtx, g.executable, method)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "error creating stdin pipe; error: %+v", err)
	}

	if _, err := stdin.Write(f.Data); err != nil {
		stdin.Close()
		return status.Errorf(codes.Internal, "error writing to stdin; error: %+v", err)
	}
	_ = stdin.Close()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return status.Errorf(codes.Internal, "process failed: %v; output=%s", err, string(out))
	}

	f = &codec.GrpcFrame{Data: out}

	if err = inStream.SendMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+v", err)
	}

	return nil
}

// grpcLoopback echoes one frame without invoking a backend service.
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

// withDefaultTimeout adds a deadline only when the caller has none and d is positive.
// It preserves an existing deadline, even if it exceeds the configured default.
func withDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok || d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
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
