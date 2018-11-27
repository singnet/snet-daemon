package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/singnet/snet-daemon/blockchain"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/gorilla/rpc/v2/json2"
	"github.com/singnet/snet-daemon/codec"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var grpcDesc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

type grpcHandler struct {
	grpcConn            *grpc.ClientConn
	enc                 string
	passthroughEndpoint string
	executable          string
}

func NewGrpcHandler() grpc.StreamHandler {
	passthroughEnabled := config.GetBool(config.PassthroughEnabledKey)

	if !passthroughEnabled {
		return grpcLoopback
	}

	h := grpcHandler{
		enc:                 blockchain.GetWireEncoding(),
		passthroughEndpoint: config.GetString(config.PassthroughEndpointKey),
		executable:          config.GetString(config.ExecutablePathKey),
	}

	switch blockchain.GetServiceType() {
	case "grpc":
		passthroughURL, err := url.Parse(h.passthroughEndpoint)
		if err != nil {
			log.WithError(err).Panic("error parsing passthrough endpoint")
		}

		conn, err := grpc.Dial(passthroughURL.Host, grpc.WithInsecure())
		if err != nil {
			log.WithError(err).Panic("error dialing service")
		}
		h.grpcConn = conn
		return h.grpcToGRPC
	case "jsonrpc":
		return h.grpcToJSONRPC
	case "process":
		return h.grpcToProcess
	}
	return nil
}

/*
Modified from https://github.com/mwitkow/grpc-proxy/blob/67591eb23c48346a480470e462289835d96f70da/proxy/handler.go#L61
Original Copyright 2017 Michal Witkowski. All Rights Reserved. See LICENSE-GRPC-PROXY for licensing terms.
Modifications Copyright 2018 SingularityNET Foundation. All Rights Reserved. See LICENSE for licensing terms.
*/
func (g grpcHandler) grpcToGRPC(srv interface{}, inStream grpc.ServerStream) error {
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
	outStream, err := g.grpcConn.NewStream(outCtx, grpcDesc, method, grpc.CallContentSubtype(g.enc))
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
			if err := src.RecvMsg(f); err != nil {
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

func (g grpcHandler) grpcToJSONRPC(srv interface{}, inStream grpc.ServerStream) error {
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

	params := new(interface{})

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

	result := new(interface{})

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

func (g grpcHandler) grpcToProcess(srv interface{}, inStream grpc.ServerStream) error {
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

func grpcLoopback(srv interface{}, inStream grpc.ServerStream) error {
	f := &codec.GrpcFrame{}
	if err := inStream.RecvMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error receiving request; error: %+v", err)
	}

	if err := inStream.SendMsg(f); err != nil {
		return status.Errorf(codes.Internal, "error sending response; error: %+v", err)
	}

	return nil
}
