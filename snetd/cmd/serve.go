package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	bolt "github.com/coreos/bbolt"
	"github.com/gorilla/handlers"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/handler"
	"github.com/singnet/snet-daemon/handler/httphandler"
	"github.com/singnet/snet-daemon/logger"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
)

var corsOptions = []handlers.CORSOption{
	handlers.AllowedHeaders([]string{"Content-Type", "Snet-Job-Address", "Snet-Job-Signature"}),
}

var ServeCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		loadConfigFileFromCommandLine(cmd.Flags().Lookup("config"))

		err = logger.InitLogger(config.SubWithDefault(config.Vip(), config.LogKey))
		if err != nil {
			log.WithError(err).Fatal("Unable to initialize logger")
		}

		var d daemon
		d, err = newDaemon()
		if err != nil {
			log.WithError(err).Fatal("Unable to initialize daemon")
		}

		d.start()
		defer d.stop()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		<-sigChan

		log.Debug("exiting")
	},
}

func loadConfigFileFromCommandLine(configFlag *pflag.Flag) {
	var err error
	var configFile = configFlag.Value.String()

	// if file is not specified by user then configFile contains default name
	if configFlag.Changed || isFileExist(configFile) {
		err = config.LoadConfig(configFile)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Fatal("Error reading configuration file")
		}
		log.WithField("configFile", configFile).Info("Using configuration file")
	} else {
		log.Info("Configuration file is not set, using default configuration")
	}
}

func isFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

type daemon struct {
	autoSSLDomain string
	acmeListener  net.Listener
	grpcServer    *grpc.Server
	blockProc     blockchain.Processor
	lis           net.Listener
	boltDB        *bolt.DB
	sslCert       *tls.Certificate
}

func newDaemon() (daemon, error) {
	d := daemon{}

	if err := config.Validate(); err != nil {
		return d, err
	}

	if config.GetBool(config.BlockchainEnabledKey) {
		if database, err := db.Connect(config.GetString(config.DbPathKey)); err != nil {
			return d, errors.Wrap(err, "unable to initialize bolt DB for blockchain state")
		} else {
			d.boltDB = database
		}
	}

	var err error
	d.lis, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%+v",
		config.GetInt(config.DaemonListeningPortKey)))
	if err != nil {
		return d, errors.Wrap(err, "error listening")
	}

	d.autoSSLDomain = config.GetString(config.AutoSSLDomainKey)
	// In order to perform the LetsEncrypt (ACME) http-01 challenge-response, we need to bind
	// port 80 (privileged) to listen for the challenge.
	if d.autoSSLDomain != "" {
		d.acmeListener, err = net.Listen("tcp", ":80")
		if err != nil {
			return d, errors.Wrap(err, "unable to bind port 80 for automatic SSL verification")
		}
	}

	d.blockProc, err = blockchain.NewProcessor(d.boltDB)
	if err != nil {
		return d, errors.Wrap(err, "unable to initialize blockchain processor")
	}

	if sslKey := config.GetString(config.SSLKeyPathKey); sslKey != "" {
		cert, err := tls.LoadX509KeyPair(config.GetString(config.SSLCertPathKey), sslKey)
		if err != nil {
			return d, errors.Wrap(err, "unable to load specifiec SSL X509 keypair")
		}
		d.sslCert = &cert
	}

	return d, nil
}

func (d daemon) start() {
	d.blockProc.StartLoop()

	var tlsConfig *tls.Config

	if d.autoSSLDomain != "" {
		log.Debug("enabling automatic SSL support")
		certMgr := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(d.autoSSLDomain),
			Cache:      autocert.DirCache(config.GetString(config.AutoSSLCacheDirKey)),
		}

		// This is the HTTP server that handles ACME challenge/response
		acmeSrv := http.Server{
			Handler: certMgr.HTTPHandler(nil),
		}
		go acmeSrv.Serve(d.acmeListener)

		tlsConfig = &tls.Config{
			GetCertificate: func(c *tls.ClientHelloInfo) (*tls.Certificate, error) {
				crt, err := certMgr.GetCertificate(c)
				if err != nil {
					log.WithError(err).Error("unable to fetch certificate")
				}
				return crt, err
			},
		}
	} else if d.sslCert != nil {
		log.Debug("enabling SSL support via X509 keypair")
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{*d.sslCert},
		}
	}

	if tlsConfig != nil {
		// See: https://gist.github.com/soheilhy/bb272c000f1987f17063
		tlsConfig.NextProtos = []string{"http/1.1", http2.NextProtoTLS, "h2-14"}

		// Wrap underlying listener with a TLS listener
		d.lis = tls.NewListener(d.lis, tlsConfig)
	}

	paymentChannelStorage := escrow.NewCombinedStorage(
		&d.blockProc,
		escrow.NewPaymentChannelStorage(escrow.NewMemStorage()),
	)

	if config.GetString(config.DaemonTypeKey) == "grpc" {
		d.grpcServer = grpc.NewServer(
			grpc.UnknownServiceHandler(handler.NewGrpcHandler()),
			grpc.StreamInterceptor(d.getGrpcInterceptor(paymentChannelStorage)),
		)
		escrow.RegisterPaymentChannelStateServiceServer(d.grpcServer, escrow.NewPaymentChannelStateService(paymentChannelStorage))

		mux := cmux.New(d.lis)
		// Use "prefix" matching to support "application/grpc*" e.g. application/grpc+proto or +json
		// Use SendSettings for compatibility with Java gRPC clients:
		//   https://github.com/soheilhy/cmux#limitations
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
		httpL := mux.Match(cmux.HTTP1Fast())

		grpcWebServer := grpcweb.WrapServer(d.grpcServer, grpcweb.WithCorsForRegisteredEndpointsOnly(false))

		httpHandler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req) {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				if strings.Split(req.URL.Path, "/")[1] == "encoding" {
					resp.Header().Set("Access-Control-Allow-Origin", "*")
					fmt.Fprintln(resp, config.GetString(config.WireEncodingKey))
				} else {
					http.NotFound(resp, req)
				}
			}
		})

		log.Debug("starting daemon")

		go d.grpcServer.Serve(grpcL)
		go http.Serve(httpL, httpHandler)
		go mux.Serve()
	} else {
		log.Debug("starting simple HTTP daemon")

		go http.Serve(d.lis, handlers.CORS(corsOptions...)(httphandler.NewHTTPHandler(d.blockProc)))
	}
}

func (d *daemon) getGrpcInterceptor(paymentChannelStorage escrow.PaymentChannelStorage) grpc.StreamServerInterceptor {
	if !d.blockProc.Enabled() {
		log.Info("Blockchain is disabled: no payment validation")
		return handler.NoOpInterceptor
	}

	log.Info("Blockchain is enabled: instantiate payment validation interceptor")
	return handler.GrpcStreamInterceptor(
		blockchain.NewJobPaymentHandler(&d.blockProc),
		escrow.NewEscrowPaymentHandler(
			&d.blockProc,
			paymentChannelStorage,
			escrow.NewIncomeValidator(&d.blockProc),
		),
	)
}

func (d daemon) stop() {
	if d.boltDB != nil {
		d.boltDB.Close()
	}

	if d.grpcServer != nil {
		d.grpcServer.Stop()
	}

	d.lis.Close()

	if d.acmeListener != nil {
		d.acmeListener.Close()
	}

	// TODO(aiden) add d.blockProc.StopLoop()
}
