package handler

import (
	"net/http"
	"net/url"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	grpcDesc            = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}
	grpcConn            *grpc.ClientConn
	enc                 string
	passthroughEndpoint string
	executable          string
	passthroughEnabled  bool
	blockchainEnabled   bool
)

func init() {
	enc = config.GetString(config.WireEncodingKey)
	passthroughEndpoint = config.GetString(config.PassthroughEndpointKey)
	executable = config.GetString(config.ExecutablePathKey)
	passthroughEnabled = config.GetBool(config.PassthroughEnabledKey)
	blockchainEnabled = config.GetBool(config.BlockchainEnabledKey)

	if config.GetString(config.ServiceTypeKey) == "grpc" && passthroughEnabled {
		passthroughUrl, err := url.Parse(passthroughEndpoint)
		if err != nil {
			log.WithError(err).Panic("error parsing passthrough endpoint")
		}

		conn, err := grpc.Dial(passthroughUrl.Host, grpc.WithInsecure())
		if err != nil {
			log.WithError(err).Panic("error dialing service")
		}
		grpcConn = conn
	}
}

func GetGrpcHandler() grpc.StreamHandler {
	if passthroughEnabled {
		switch config.GetString(config.ServiceTypeKey) {
		case "grpc":
			return grpcToGrpc
		case "jsonrpc":
			return grpcToJsonRpc
		case "process":
			return grpcToProcess
		}
		return nil
	}

	return grpcLoopback
}

func GetHttpHandler(bp blockchain.Processor) http.Handler {
	return httpToHttp(bp)
}
