package cmd

import (
	"os"

	"github.com/coreos/bbolt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/singnet/snet-daemon/handler"
)

type Components struct {
	db                         *bbolt.DB
	blockchain                 *blockchain.Processor
	etcdClient                 *etcddb.EtcdClient
	etcdServer                 *etcddb.EtcdServer
	paymentChannelStorage      escrow.PaymentChannelStorage
	grpcInterceptor            grpc.StreamServerInterceptor
	paymentChannelStateService *escrow.PaymentChannelStateService
}

func InitComponents(cmd *cobra.Command) (components *Components, err error) {
	components = &Components{}
	defer func() {
		if err != nil {
			components.Close()
			components = nil
		}
	}()

	loadConfigFileFromCommandLine(cmd.Flags().Lookup("config"))

	for _, init := range []func() error{
		components.initPaymentChannelStorage,
		components.initGrpcInterceptor,
		components.initPaymentChannelStateService,
	} {
		err = init()
		if err != nil {
			return
		}
	}

	return
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

func (components *Components) initPaymentChannelStorage() (err error) {
	var delegateStorage escrow.AtomicStorage
	if config.GetString(config.PaymentChannelStorageTypeKey) == "etcd" {
		client, err := etcddb.NewEtcdClient()
		if err != nil {
			return errors.Wrap(err, "unable to create etcd client")
		}

		components.etcdClient = client
		delegateStorage = client
	} else {
		delegateStorage = escrow.NewMemStorage()
	}

	components.paymentChannelStorage = escrow.NewCombinedStorage(
		components.Blockchain(),
		escrow.NewPaymentChannelStorage(delegateStorage),
	)

	return nil
}

func (components *Components) initGrpcInterceptor() (err error) {
	if !components.Blockchain().Enabled() {
		log.Info("Blockchain is disabled: no payment validation")
		components.grpcInterceptor = handler.NoOpInterceptor
		return nil
	}

	log.Info("Blockchain is enabled: instantiate payment validation interceptor")
	components.grpcInterceptor = handler.GrpcStreamInterceptor(
		blockchain.NewJobPaymentHandler(components.Blockchain()),
		escrow.NewEscrowPaymentHandler(
			components.Blockchain(),
			components.paymentChannelStorage,
			escrow.NewIncomeValidator(components.Blockchain()),
		),
	)
	return nil
}

func (components *Components) initPaymentChannelStateService() (err error) {
	components.paymentChannelStateService = escrow.NewPaymentChannelStateService(components.PaymentChannelStorage())
	return nil
}

func (components *Components) EtcdServer() (server *etcddb.EtcdServer) {
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

	server, err = etcddb.GetEtcdServer()
	if err != nil {
		log.WithError(err).Panic("error during etcd config parsing")
	}

	components.etcdServer = server
	return server
}

func (components *Components) Close() {
	if components.db != nil {
		components.db.Close()
	}
	if components.etcdClient != nil {
		components.etcdClient.Close()
	}
	if components.etcdServer != nil {
		components.etcdServer.Close()
	}
}

func (components *Components) DB() *bbolt.DB {
	if components.db != nil {
		return components.db
	}

	if config.GetBool(config.BlockchainEnabledKey) {
		if database, err := db.Connect(config.GetString(config.DbPathKey)); err != nil {
			log.WithError(err).Panic("unable to initialize bbolt DB for blockchain state")
		} else {
			components.db = database
		}
	}

	return components.db
}

func (components *Components) Blockchain() *blockchain.Processor {
	if components.blockchain != nil {
		return components.blockchain
	}

	processor, err := blockchain.NewProcessor(components.DB())
	if err != nil {
		log.WithError(err).Panic("unable to initialize blockchain processor")
	}

	components.blockchain = &processor
	return components.blockchain
}

func (components *Components) PaymentChannelStorage() (storage escrow.PaymentChannelStorage) {
	return components.paymentChannelStorage
}

func (components *Components) GrpcInterceptor() (interceptor grpc.StreamServerInterceptor) {
	return components.grpcInterceptor
}

func (components *Components) PaymentChannelStateService() (service *escrow.PaymentChannelStateService) {
	return components.paymentChannelStateService
}
