package cmd

import (
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/singnet/snet-daemon/metrics"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/singnet/snet-daemon/handler"
)

type Components struct {
	serviceMetadata            *blockchain.ServiceMetadata
	blockchain                 *blockchain.Processor
	etcdClient                 *etcddb.EtcdClient
	etcdServer                 *etcddb.EtcdServer
	atomicStorage              escrow.AtomicStorage
	paymentChannelService      escrow.PaymentChannelService
	escrowPaymentHandler       handler.PaymentHandler
	grpcInterceptor            grpc.StreamServerInterceptor
	paymentChannelStateService *escrow.PaymentChannelStateService
	etcdLockerStorage          *escrow.PrefixedAtomicStorage
	providerControlService     *escrow.ProviderControlService
	daemonHeartbeat            *metrics.DaemonHeartbeat
}

func InitComponents(cmd *cobra.Command) (components *Components) {
	components = &Components{}
	defer func() {
		err := recover()
		if err != nil {
			components.Close()
			components = nil
			panic("re-panic after components cleanup")
		}
	}()

	loadConfigFileFromCommandLine(cmd.Flags().Lookup("config"))

	return
}

func loadConfigFileFromCommandLine(configFlag *pflag.Flag) {
	var configFile = configFlag.Value.String()

	// if file is not specified by user then configFile contains default name
	if configFlag.Changed || isFileExist(configFile) {
		err := config.LoadConfig(configFile)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Panic("Error reading configuration file")
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

func (components *Components) Close() {
	if components.etcdClient != nil {
		components.etcdClient.Close()
	}
	if components.etcdServer != nil {
		components.etcdServer.Close()
	}
	if components.blockchain != nil {
		components.blockchain.Close()
	}
}

func (components *Components) Blockchain() *blockchain.Processor {
	if components.blockchain != nil {
		return components.blockchain
	}

	processor, err := blockchain.NewProcessor(components.ServiceMetaData())
	if err != nil {
		log.WithError(err).Panic("unable to initialize blockchain processor")
	}

	components.blockchain = &processor
	return components.blockchain
}

func (components *Components) ServiceMetaData() *blockchain.ServiceMetadata {
	if components.serviceMetadata != nil {
		return components.serviceMetadata
	}
	components.serviceMetadata = blockchain.ServiceMetaData()
	return components.serviceMetadata
}

func (components *Components) EtcdServer() *etcddb.EtcdServer {
	if components.etcdServer != nil {
		return components.etcdServer
	}

	enabled, err := etcddb.IsEtcdServerEnabled()
	if err != nil {
		log.WithError(err).Panic("error during etcd config parsing")
	}
	if !enabled {
		return nil
	}

	server, err := etcddb.GetEtcdServer()
	if err != nil {
		log.WithError(err).Panic("error during etcd config parsing")
	}

	err = server.Start()
	if err != nil {
		log.WithError(err).Panic("error during etcd server starting")
	}

	components.etcdServer = server
	return server
}

func (components *Components) EtcdClient() *etcddb.EtcdClient {
	if components.etcdClient != nil {
		return components.etcdClient
	}

	client, err := etcddb.NewEtcdClient()
	if err != nil {
		log.WithError(err).Panic("unable to create etcd client")
	}

	components.etcdClient = client
	return components.etcdClient
}

func (components *Components) LockerStorage() *escrow.PrefixedAtomicStorage {
	if components.etcdLockerStorage != nil {
		return components.etcdLockerStorage
	}
	components.etcdLockerStorage = escrow.NewLockerStorage(components.AtomicStorage())
	return components.etcdLockerStorage
}

func (components *Components) AtomicStorage() escrow.AtomicStorage {
	if components.atomicStorage != nil {
		return components.atomicStorage
	}

	if config.GetString(config.PaymentChannelStorageTypeKey) == "etcd" {
		components.atomicStorage = components.EtcdClient()
	} else {
		components.atomicStorage = escrow.NewMemStorage()
	}

	return components.atomicStorage
}

func (components *Components) PaymentChannelService() escrow.PaymentChannelService {
	if components.paymentChannelService != nil {
		return components.paymentChannelService
	}

	components.paymentChannelService = escrow.NewPaymentChannelService(
		escrow.NewPaymentChannelStorage(components.AtomicStorage()),
		escrow.NewPaymentStorage(components.AtomicStorage()),
		escrow.NewBlockchainChannelReader(components.Blockchain(), config.Vip(), components.ServiceMetaData()),
		escrow.NewEtcdLocker(components.AtomicStorage()),
		escrow.NewChannelPaymentValidator(components.Blockchain(), config.Vip(), components.ServiceMetaData()),func() ([32]byte, error) {
			s := components.ServiceMetaData().GetDaemonGroupID()
			return s, nil
		},
	)

	return components.paymentChannelService
}

func (components *Components) EscrowPaymentHandler() handler.PaymentHandler {
	if components.escrowPaymentHandler != nil {
		return components.escrowPaymentHandler
	}

	components.escrowPaymentHandler = escrow.NewPaymentHandler(
		components.PaymentChannelService(),
		components.Blockchain(),
		escrow.NewIncomeValidator(components.ServiceMetaData().GetPriceInCogs()),
	)

	return components.escrowPaymentHandler
}

//Add a chain of interceptors
func (components *Components) GrpcInterceptor() grpc.StreamServerInterceptor {
	if components.grpcInterceptor != nil {
		return components.grpcInterceptor
	}
	//If monitoring is enabled and the endpoint URL is valid and if the
	// Daemon has successfully registered itself and has obtained a valid token to publish metrics
	// , ONLY then add this interceptor to the chain of interceptors
	metrics.SetDaemonGrpId(components.ServiceMetaData().GetDaemonGroupIDString())
	if config.GetBool(config.MonitoringEnabled) &&
		config.IsValidUrl(config.GetString(config.MonitoringServiceEndpoint)) &&
		metrics.RegisterDaemon(config.GetString(config.MonitoringServiceEndpoint)+"/register") {

		components.grpcInterceptor = grpc_middleware.ChainStreamServer(
			handler.GrpcMonitoringInterceptor(), handler.GrpcRateLimitInterceptor(),
			components.GrpcPaymentValidationInterceptor())
	} else {
		components.grpcInterceptor = grpc_middleware.ChainStreamServer(handler.GrpcRateLimitInterceptor(),
			components.GrpcPaymentValidationInterceptor())
	}
	return components.grpcInterceptor
}

func (components *Components) GrpcPaymentValidationInterceptor() grpc.StreamServerInterceptor {
	if !components.Blockchain().Enabled() {
		log.Info("Blockchain is disabled: no payment validation")
		return handler.NoOpInterceptor
	} else {
		log.Info("Blockchain is enabled: instantiate payment validation interceptor")
		return handler.GrpcPaymentValidationInterceptor(components.EscrowPaymentHandler())
	}
}

func (components *Components) PaymentChannelStateService() (service *escrow.PaymentChannelStateService) {
	if components.paymentChannelStateService != nil {
		return components.paymentChannelStateService
	}

	components.paymentChannelStateService = escrow.NewPaymentChannelStateService(
		components.PaymentChannelService(),
		escrow.NewPaymentStorage(components.AtomicStorage()),)

	return components.paymentChannelStateService
}

//NewProviderControlService

func (components *Components) ProviderControlService() (service *escrow.ProviderControlService) {
	if components.providerControlService != nil {
		return components.providerControlService
	}

	components.providerControlService = escrow.NewProviderControlService(components.PaymentChannelService(),components.ServiceMetaData())
	return components.providerControlService
}

func (components *Components) DaemonHeartBeat() (service *metrics.DaemonHeartbeat) {
	if components.daemonHeartbeat != nil {
		return components.daemonHeartbeat
	}
	metrics.SetDaemonGrpId(components.ServiceMetaData().GetDaemonGroupIDString())
	components.daemonHeartbeat = &metrics.DaemonHeartbeat{DaemonID:metrics.GetDaemonID()}
	return components.daemonHeartbeat
}
