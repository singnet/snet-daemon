package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	"github.com/singnet/snet-daemon/handler"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

func main() {
	log.SetLevel(log.Level(config.GetInt(config.LogLevelKey)))
	config.Validate()

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%+v",
		config.GetInt(config.DaemonListeningPortKey)))
	if err != nil {
		log.WithError(err).Panic("error listening")
	}

	if config.GetString(config.DaemonTypeKey) == "grpc" {
		mux := cmux.New(lis)
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type",
			"application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		grpcServer := grpc.NewServer(grpc.UnknownServiceHandler(handler.GetGrpcHandler()),
			grpc.StreamInterceptor(blockchain.GetGrpcStreamInterceptor()))
		grpcWebServer := grpcweb.WrapServer(grpcServer)

		log.Debug("starting daemon")

		go grpcServer.Serve(grpcL)
		go http.Serve(httpL, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(req) {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				if strings.Split(req.URL.Path, "/")[1] == "encoding" {
					fmt.Fprint(resp, config.GetString(config.WireEncodingKey))
				} else {
					http.NotFound(resp, req)
				}
			}
		}))
		go mux.Serve()

		defer grpcServer.Stop()
	} else {
		log.Debug("starting daemon")

		go http.Serve(lis, handler.GetHttpHandler())
	}

	defer db.Shutdown()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Debug("exiting")
}
