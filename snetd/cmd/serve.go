package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/configuration_service"
	contractListener "github.com/singnet/snet-daemon/v6/contract_event_listener"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/freecall"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/handler/httphandler"
	"github.com/singnet/snet-daemon/v6/lifecycle"
	"github.com/singnet/snet-daemon/v6/logger"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

var corsOptionsHTTP = []handlers.CORSOption{
	handlers.AllowedHeaders([]string{"*"}),
	handlers.AllowedOrigins([]string{"*"}),
	handlers.ExposedHeaders([]string{"*"}),
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Is the default option which starts the Daemon.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		components := InitComponents(cmd)
		defer components.Close()

		validationReport := validateDaemonConfiguration()
		if validationReport.HasErrors() {
			cmd.SilenceUsage = true
			source := config.Vip().ConfigFileUsed()
			if source == "" {
				source = "defaults"
			}
			if err := writeConfigValidationReport(cmd.ErrOrStderr(), "text", source, validationReport); err != nil {
				return err
			}
			return ErrInvalidConfiguration
		}

		logger.Initialize()
		for _, diagnostic := range validationReport.Diagnostics {
			if diagnostic.Severity == config.DiagnosticWarning {
				zap.L().Warn(diagnostic.Message, zap.String("code", diagnostic.Code), zap.String("field", diagnostic.Field))
			}
		}

		config.LogConfig()
		if message, err := config.CheckVersionOfDaemon(); err != nil {
			zap.L().Warn(err.Error())
		} else {
			zap.L().Info(message)
		}

		etcdServer := components.EtcdServer()
		if etcdServer != nil {
			zap.L().Info("Using internal etcd server because it is enabled in config")
		}

		d, err := newDaemon(components)
		if err != nil {
			zap.L().Fatal(fmt.Sprintf("Unable to initialize daemon: %v %v ", err, errs.ErrDescURL(errs.InvalidConfig)))
		}

		d.serve()
		return nil
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
	requests      *lifecycle.Group
	httpServers   []*http.Server
	listeners     []net.Listener
	connections   daemonConnections
	servers       sync.WaitGroup
	stopOnce      sync.Once
	stopping      atomic.Bool
	closeProxy    func()
}

func (d *daemon) serve() {
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopSignals()

	d.start()
	defer d.stop()

	// With an initialized, hot-reload-enabled etcd client, watch Registry
	// organization updates and reconnect when the metadata's etcd endpoints change.
	if d.components.etcdClient != nil && d.components.etcdClient.IsHotReloadEnabled() {
		contractEventListener := contractListener.ContractEventListener{
			BlockchainProcessor:         d.blockProc,
			CurrentOrganizationMetaData: d.components.OrganizationMetaData(),
			CurrentEtcdClient:           d.components.EtcdClient(),
		}
		d.components.Background().Go(contractEventListener.ListenOrganizationMetadataChangingContext)
	}

	<-ctx.Done()
	zap.L().Debug("Exiting")
}

func newDaemon(components *Components) (*daemon, error) {
	d := &daemon{}
	if err := validateDaemonConfiguration().Error(); err != nil {
		return d, err
	}

	d.components = components

	d.blockProc = components.Blockchain()

	exp := config.GetExperimentalSettings()
	if exp != nil && exp.TrafficSplit != nil {
		zap.L().Debug("using experimental settings:", zap.Any("parsed", exp))
		return d, nil
	}

	var err error
	d.lis, err = net.Listen("tcp", config.GetString(config.DaemonEndpoint))
	if err != nil {
		return d, errors.Wrap(err, "Expected format of daemon_endpoint is <host>:<port>.Error binding to the endpoint:"+config.GetString(config.DaemonEndpoint))
	}

	d.autoSSLDomain = config.GetString(config.AutoSSLDomainKey)
	// ACME HTTP-01 domain validation requires a listener on port 80.
	if d.autoSSLDomain != "" {
		d.acmeListener, err = net.Listen("tcp", ":80")
		if err != nil {
			return d, errors.Wrap(err, "unable to bind port 80 for automatic SSL verification")
		}
	}

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
	if d.components == nil {
		d.components = &Components{}
	}
	d.requests = lifecycle.New()
	if d.lis != nil {
		d.lis = d.ownListener(d.lis)
	}

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
		d.components.Background().Go(certReloader.ListenContext)
	}

	if d.autoSSLDomain != "" {
		zap.L().Debug("enabling automatic SSL support")
		certMgr := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(d.autoSSLDomain),
			Cache:      autocert.DirCache(config.GetString(config.AutoSSLCacheDirKey)),
		}

		// Serve ACME HTTP-01 challenges separately from the daemon listener.
		d.acmeListener = d.ownListener(d.acmeListener)
		d.serveHTTP(d.acmeListener, certMgr.HTTPHandler(nil))

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
		// Advertise HTTP/1.1 and HTTP/2 so HTTP and native gRPC can share the TLS listener.
		tlsConfig.NextProtos = []string{"http/1.1", http2.NextProtoTLS, "h2-14"}

		// Wrap underlying listener with a TLS listener
		d.lis = tls.NewListener(d.lis, tlsConfig)
	}

	if config.GetString(config.DaemonTypeKey) != "grpc" {
		zap.L().Debug("starting simple HTTP daemon")
		proxy := httphandler.NewHTTPHandler(d.blockProc)
		d.closeProxy = proxy.Close
		d.serveHTTP(d.lis, handlers.CORS(corsOptionsHTTP...)(proxy))
		return
	}

	maxsizeOpt := grpc.MaxRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB) * 1024 * 1024)
	proxy, cleanup := handler.NewGrpcHandlerWithClose(d.components.ServiceMetaData())
	d.closeProxy = cleanup
	options := []grpc.ServerOption{
		grpc.UnknownServiceHandler(proxy),
		grpc.ChainStreamInterceptor(d.trackStream, d.components.GrpcStreamInterceptor()),
		maxsizeOpt,
	}
	if unary := d.components.GrpcUnaryInterceptor(); unary != nil {
		options = append(options, grpc.ChainUnaryInterceptor(d.trackUnary, unary))
	} else {
		options = append(options, grpc.UnaryInterceptor(d.trackUnary))
	}
	d.grpcServer = grpc.NewServer(options...)
	escrow.RegisterPaymentChannelStateServiceServer(d.grpcServer, d.components.PaymentChannelStateService())
	escrow.RegisterProviderControlServiceServer(d.grpcServer, d.components.ProviderControlService())
	freecall.RegisterFreeCallStateServiceServer(d.grpcServer, d.components.FreeCallStateService())
	escrow.RegisterTokenServiceServer(d.grpcServer, d.components.TokenService())
	training.RegisterDaemonServer(d.grpcServer, d.components.TrainingService())
	grpc_health_v1.RegisterHealthServer(d.grpcServer, d.components.DaemonHeartBeat())
	configuration_service.RegisterConfigurationServiceServer(d.grpcServer, d.components.ConfigurationService())

	var gmux GRPCMux

	exp := config.GetExperimentalSettings()
	if exp == nil {
		exp = &config.ExperimentalSettings{
			SplitWebgrpc:    false,
			UseOriginalCmux: false,
			TrafficSplit:    nil,
		}
	}
	if exp.TrafficSplit != nil {
		d.startWithTrafficSplit(exp)
		return
	}

	if exp.UseOriginalCmux {
		gmux = newOriginalMux(d.lis, exp.SplitWebgrpc)
	} else {
		gmux = newForkMux(d.lis, exp.SplitWebgrpc)
	}

	endpoints := gmux.Endpoints()

	grpcWebServer := d.newGRPCWebServer()
	httpHandler := d.newHTTPHandler(grpcWebServer)
	corsOpts := handler.Cors()

	for _, ep := range endpoints {
		switch ep.Type {
		case L_GRPC:
			d.runServer(func() error { return d.grpcServer.Serve(ep.L) })
		case L_GRPC_WEB:
			d.serveHTTP(ep.L, corsOpts.Handler(grpcWebServer))
		case L_HTTP:
			d.serveHTTP(ep.L, corsOpts.Handler(httpHandler))
		}
	}

	d.runServer(gmux.Serve)

	zap.L().Info("✅ Daemon successfully started and ready to accept requests")
}

// startWithTrafficSplit starts separate listeners for gRPC and HTTP
// instead of using cmux. This mode is intended for setups where
// L7 proxies (nginx/ingress/traefik) already split traffic by port.
// These listeners are plain TCP; any required TLS termination must happen upstream.
func (d *daemon) startWithTrafficSplit(exp *config.ExperimentalSettings) {
	ts := exp.TrafficSplit

	// This warning does not apply TLS to the separate listeners below.
	if d.sslCert != nil || d.autoSSLDomain != "" {
		zap.L().Warn("traffic_split mode is enabled, but TLS is also configured on daemon side; make sure this is really what you want")
	}

	host, _, err := net.SplitHostPort(config.GetString(config.DaemonEndpoint))
	if err != nil || host == "" {
		// If DaemonEndpoint is just ':PORT' or invalid, bind on all interfaces.
		host = ""
	}

	grpcAddr := fmt.Sprintf("%s:%d", host, ts.Grpc)
	httpAddr := fmt.Sprintf("%s:%d", host, ts.Http)

	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		zap.L().Fatal("failed to listen in traffic_split mode for gRPC", zap.String("addr", grpcAddr), zap.Error(err))
	}

	httpLis, err := net.Listen("tcp", httpAddr)
	if err != nil {
		zap.L().Fatal("failed to listen in traffic_split mode for HTTP", zap.String("addr", httpAddr), zap.Error(err))
	}

	grpcWebServer := d.newGRPCWebServer()
	httpHandler := d.newHTTPHandler(grpcWebServer)
	corsOpts := handler.Cors()

	if d.requests == nil {
		d.requests = lifecycle.New()
	}
	grpcLis, httpLis = d.ownListener(grpcLis), d.ownListener(httpLis)
	d.runServer(func() error { return d.grpcServer.Serve(grpcLis) })
	d.serveHTTP(httpLis, corsOpts.Handler(httpHandler))

	zap.L().Info("✅ Daemon started in traffic_split mode",
		zap.String("grpc_addr", grpcAddr),
		zap.String("http_addr", httpAddr),
	)
}

// newGRPCWebServer wraps the gRPC server with grpc-web support.
func (d *daemon) newGRPCWebServer() *grpcweb.WrappedGrpcServer {
	return grpcweb.WrapServer(
		d.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)
}

// newHTTPHandler serves gRPC-Web, encoding, and heartbeat routes in both cmux
// and traffic_split modes. HTTP routes use the first path component; other
// requests return 404. Callers wrap it with the CORS handler.
func (d *daemon) newHTTPHandler(grpcWebServer *grpcweb.WrappedGrpcServer) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// The outer CORS handler normally handles preflight. This fallback also
		// returns 204 when this handler is used without that wrapper.
		if req.Method == http.MethodOptions {
			zap.L().Debug("[options] manual options",
				zap.String("path", req.URL.Path),
				zap.String("method", req.Method),
			)
			resp.WriteHeader(http.StatusNoContent)
			return
		}

		// Dispatch gRPC-Web before interpreting the path as a utility endpoint.
		if grpcWebServer != nil && grpcWebServer.IsGrpcWebRequest(req) {
			zap.L().Debug("[grpc-web]",
				zap.String("path", req.URL.Path),
				zap.String("method", req.Method),
			)
			grpcWebServer.ServeHTTP(resp, req)
			return
		}

		// Simple HTTP endpoints (encoding / heartbeat)
		var path string
		if parts := strings.Split(req.URL.Path, "/"); len(parts) > 1 {
			path = parts[1]
		}
		switch path {
		case "encoding":
			fmt.Fprintln(resp, d.components.ServiceMetaData().GetWireEncoding())
		case "heartbeat":
			metrics.HeartbeatHandlerContext(req.Context(), resp,
				func() (*training.TrainingMetadata, error) {
					return d.components.TrainingService().GetTrainingMetadata(
						req.Context(), &emptypb.Empty{},
					)
				},
				d.components.DaemonHeartBeat().DynamicPricing,
				d.components.Blockchain().CurrentBlock,
				d.components.Background(),
			)
		default:
			http.NotFound(resp, req)
			return
		}

		zap.L().Debug("http headers", zap.Any("headers", resp.Header()))
	})
}
