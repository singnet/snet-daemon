package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/singnet/snet-daemon/v6/errs"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/configuration_service"
	contractListener "github.com/singnet/snet-daemon/v6/contract_event_listener"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/handler/httphandler"
	"github.com/singnet/snet-daemon/v6/logger"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/singnet/snet-daemon/v6/training"

	"github.com/gorilla/handlers"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/semyon-dev/cmux"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var corsOptions = []handlers.CORSOption{
	handlers.AllowedHeaders([]string{"Content-Type", "Snet-Job-Address", "Snet-Job-Signature"}),
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Is the default option which starts the Daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		components := InitComponents(cmd)
		defer components.Close()

		logger.Initialize()
		config.LogConfig()

		etcdServer := components.EtcdServer()
		if etcdServer != nil {
			zap.L().Info("Using internal etcd server because it is enabled in config")
		}

		var d daemon
		d, err = newDaemon(components)
		if err != nil {
			zap.L().Fatal(fmt.Sprintf("Unable to initialize daemon: %v %v ", err, errs.ErrDescURL(errs.InvalidConfig)))
		}

		d.start()
		defer d.stop()

		// Check if the payment storage client is etcd by verifying if d.components.etcdClient exists.
		// If etcdClient is not nil and hot reload is enabled, initialize a ContractEventListener
		// to listen for changes in the organization metadata.
		if d.components.etcdClient != nil && d.components.etcdClient.IsHotReloadEnabled() {
			contractEventLister := contractListener.ContractEventListener{
				BlockchainProcessor:         d.blockProc,
				CurrentOrganizationMetaData: components.OrganizationMetaData(),
				CurrentEtcdClient:           components.EtcdClient(),
			}
			go contractEventLister.ListenOrganizationMetadataChanging()
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		<-sigChan

		zap.L().Debug("Exiting")
	},
}

type daemon struct {
	autoSSLDomain string
	acmeListener  net.Listener
	grpcServer    *grpc.Server
	blockProc     blockchain.Processor
	lis           net.Listener
	sslCert       *tls.Certificate
	components    *Components
}

func newDaemon(components *Components) (daemon, error) {
	d := daemon{}

	if err := config.Validate(); err != nil {
		return d, err
	}

	// validate heartbeat configuration
	if err := metrics.ValidateHeartbeatConfig(config.GetString(config.ServiceHeartbeatType), config.GetString(config.HeartbeatServiceEndpoint)); err != nil {
		return d, err
	}

	// validate alerts/notifications configuration
	if err := metrics.ValidateNotificationConfig(); err != nil {
		return d, err
	}

	d.components = components

	var err error
	d.lis, err = net.Listen("tcp", config.GetString(config.DaemonEndpoint))
	if err != nil {
		return d, errors.Wrap(err, "Expected format of daemon_endpoint is <host>:<port>.Error binding to the endpoint:"+config.GetString(config.DaemonEndpoint))
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

	d.blockProc = components.Blockchain()

	if sslKey := config.GetString(config.SSLKeyPathKey); sslKey != "" {
		cert, err := tls.LoadX509KeyPair(config.GetString(config.SSLCertPathKey), sslKey)
		if err != nil {
			return d, errors.Wrap(err, "unable to load specific SSL X509 keypair")
		}
		d.sslCert = &cert
	}

	return d, nil
}

func (d *daemon) start() {

	var tlsConfig *tls.Config
	var certReloader *CertReloader

	if config.GetString(config.SSLCertPathKey) != "" {
		certReloader = &CertReloader{
			CertFile: config.GetString(config.SSLCertPathKey),
			KeyFile:  config.GetString(config.SSLKeyPathKey),
			mutex:    new(sync.Mutex),
		}
	}

	if certReloader != nil {
		certReloader.Listen()
	}

	if d.autoSSLDomain != "" {
		zap.L().Debug("enabling automatic SSL support")
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
					zap.L().Error("unable to fetch certificate", zap.Error(err))
				}
				return crt, err
			},
		}
	} else if d.sslCert != nil {
		zap.L().Debug("enabling SSL support via X509 keypair")
		tlsConfig = &tls.Config{
			GetCertificate: func(c *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return certReloader.GetCertificate(), nil
			},
			Certificates: []tls.Certificate{*d.sslCert},
		}
	}

	if tlsConfig != nil {
		// See: https://gist.github.com/soheilhy/bb272c000f1987f17063
		tlsConfig.NextProtos = []string{"http/1.1", http2.NextProtoTLS, "h2-14"}

		// Wrap underlying listener with a TLS listener
		d.lis = tls.NewListener(d.lis, tlsConfig)
	}

	if config.GetString(config.DaemonTypeKey) == "grpc" {

		maxsizeOpt := grpc.MaxRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB) * 1024 * 1024)
		d.grpcServer = grpc.NewServer(
			grpc.UnknownServiceHandler(handler.NewGrpcHandler(d.components.ServiceMetaData())),
			grpc.StreamInterceptor(d.components.GrpcStreamInterceptor()),
			grpc.UnaryInterceptor(d.components.GrpcUnaryInterceptor()),
			maxsizeOpt,
		)
		escrow.RegisterPaymentChannelStateServiceServer(d.grpcServer, d.components.PaymentChannelStateService())
		escrow.RegisterProviderControlServiceServer(d.grpcServer, d.components.ProviderControlService())
		escrow.RegisterFreeCallStateServiceServer(d.grpcServer, d.components.FreeCallStateService())
		escrow.RegisterTokenServiceServer(d.grpcServer, d.components.TokenService())
		training.RegisterDaemonServer(d.grpcServer, d.components.TrainingService())
		grpc_health_v1.RegisterHealthServer(d.grpcServer, d.components.DaemonHeartBeat())
		configuration_service.RegisterConfigurationServiceServer(d.grpcServer, d.components.ConfigurationService())
		mux := cmux.New(d.lis)
		// Use "prefix" matching to support "application/grpc*" e.g. application/grpc+proto or +json
		// Use SendSettings for compatibility with Java gRPC clients:
		// https://github.com/soheilhy/cmux#limitations

		// gRPC-Web (HTTP/1.1)
		grpcWebL := mux.Match(cmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web"))

		// true gRPC (HTTP/2)
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))

		// HTTP/REST
		httpL := mux.Match(cmux.HTTP1Fast())

		grpcWebServer := grpcweb.WrapServer(d.grpcServer, grpcweb.WithCorsForRegisteredEndpointsOnly(false), grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}))

		httpHandler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			isGrpcWebReq := grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req)
			zap.L().Info("http request", zap.Bool("isGrpcWebRequest", isGrpcWebReq), zap.String("path", req.URL.Path), zap.String("method", req.Method))
			resp.Header().Set("Access-Control-Allow-Origin", "*")
			if isGrpcWebReq {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				var path string
				if parts := strings.Split(req.URL.Path, "/"); len(parts) > 1 {
					path = parts[1]
				}
				switch path {
				case "encoding":
					fmt.Fprintln(resp, d.components.ServiceMetaData().GetWireEncoding())
				case "heartbeat":
					metrics.HeartbeatHandler(resp,
						func() (*training.TrainingMetadata, error) {
							return d.components.TrainingService().GetTrainingMetadata(context.Background(), &emptypb.Empty{})
						},
						d.components.DaemonHeartBeat().DynamicPricing,
						d.components.Blockchain().CurrentBlock)
				default:
					http.NotFound(resp, req)
					return
				}
			}
			zap.L().Debug("http headers", zap.Any("headers", resp.Header()))
		})

		corsOpts := cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodHead,
				http.MethodConnect,
			},
			AllowCredentials: true,
			Debug:            "debug" == config.GetString(logger.LogLevelKey),
			AllowOriginRequestFunc: func(r *http.Request, origin string) bool {
				return true
			},
			AllowOriginFunc: func(origin string) bool {
				return true
			},
			ExposedHeaders: []string{"X-Grpc-Web", "Content-Length", "Access-Control-Allow-Origin", "Content-Type", "Origin", "Grpc-Status", "Grpc-Message"},
			AllowedHeaders: []string{"Grpc-Status", "Grpc-Message", "X-Grpc-Web", "User-Agent", "Origin", "Accept", "Authorization", "Content-Type", "X-Requested-With", "Content-Length", "Access-Control-Allow-Origin",
				handler.PaymentTypeHeader,
				handler.ClientTypeHeader,
				handler.PaymentChannelSignatureHeader,
				handler.PaymentChannelIDHeader,
				handler.PaymentChannelAmountHeader,
				handler.PaymentChannelNonceHeader,
				handler.FreeCallUserIdHeader,
				handler.FreeCallUserAddressHeader,
				handler.FreeCallAuthTokenHeader,
				handler.UserInfoHeader,
				handler.UserAgentHeader,
				handler.DynamicPriceDerived,
				handler.PrePaidAuthTokenHeader,
				handler.CurrentBlockNumberHeader,
				handler.PaymentMultiPartyEscrowAddressHeader,
				handler.TrainingModelId,
			},
		})

		go d.grpcServer.Serve(grpcL)                        // HTTP/2 gRPC
		go d.grpcServer.Serve(grpcWebL)                     // HTTP/1.1 gRPC-Web
		go http.Serve(httpL, corsOpts.Handler(httpHandler)) // HTTP
		go mux.Serve()

	} else {
		zap.L().Debug("starting simple HTTP daemon")
		go http.Serve(d.lis, handlers.CORS(corsOptions...)(httphandler.NewHTTPHandler(d.blockProc)))
	}

	zap.L().Info("✅ Daemon successfully started and ready to accept requests")
}

func (d *daemon) stop() {

	if d.grpcServer != nil {
		d.grpcServer.GracefulStop()
	}

	d.lis.Close()

	if d.acmeListener != nil {
		d.acmeListener.Close()
	}

	// TODO(aiden) add d.blockProc.StopLoop()
}
