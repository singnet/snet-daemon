package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	"github.com/singnet/snet-daemon/handler"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var ServeCmd = &cobra.Command{
	Use: "snetd",
	Run: func(cmd *cobra.Command, args []string) {
		d, err := newDaemon()
		if err != nil {
			log.WithError(err).Error("Unable to initialize daemon")
			os.Exit(2)
		}

		d.start()
		defer d.stop()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		<-sigChan

		log.Debug("exiting")
	},
}

type daemon struct {
	grpcServer *grpc.Server
	blockProc  blockchain.Processor
	lis        net.Listener
}

func newDaemon() (daemon, error) {
	d := daemon{}

	if err := config.Validate(); err != nil {
		return d, err
	}

	var err error
	d.lis, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%+v",
		config.GetInt(config.DaemonListeningPortKey)))
	if err != nil {
		return d, errors.Wrap(err, "error listening")
	}

	d.blockProc, err = blockchain.NewProcessor()
	if err != nil {
		return d, errors.Wrap(err, "unable to initialize blockchain processor")
	}

	return d, nil
}

func (d daemon) start() {
	d.blockProc.StartLoop()

	if config.GetString(config.DaemonTypeKey) == "grpc" {
		mux := cmux.New(d.lis)
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type",
			"application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		d.grpcServer = grpc.NewServer(grpc.UnknownServiceHandler(handler.GetGrpcHandler()),
			grpc.StreamInterceptor(d.blockProc.GrpcStreamInterceptor()))
		grpcWebServer := grpcweb.WrapServer(d.grpcServer)

		log.Debug("starting daemon")

		go d.grpcServer.Serve(grpcL)
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
	} else {
		log.Debug("starting daemon")

		go http.Serve(d.lis, handler.GetHTTPHandler(d.blockProc))
	}
}

func (d daemon) stop() {
	db.Shutdown()

	if d.grpcServer != nil {
		d.grpcServer.Stop()
	}

	// TODO(aiden) add d.blockProc.StopLoop()
}
