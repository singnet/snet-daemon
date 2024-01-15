package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/singnet/snet-daemon/configuration_service"
	"github.com/singnet/snet-daemon/metrics"
	"github.com/singnet/snet-daemon/pricing"
	"github.com/singnet/snet-daemon/storage"
	"github.com/singnet/snet-daemon/token"
	"github.com/singnet/snet-daemon/training"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/singnet/snet-daemon/handler"
)

type Components struct {
	allowedUserPaymentHandler  handler.PaymentHandler
	serviceMetadata            *blockchain.ServiceMetadata
	blockchain                 *blockchain.Processor
	etcdClient                 *etcddb.EtcdClient
	etcdServer                 *etcddb.EtcdServer
	atomicStorage              storage.AtomicStorage
	paymentChannelService      escrow.PaymentChannelService
	escrowPaymentHandler       handler.PaymentHandler
	grpcInterceptor            grpc.StreamServerInterceptor
	paymentChannelStateService *escrow.PaymentChannelStateService
	etcdLockerStorage          *storage.PrefixedAtomicStorage
	mpeSpecificStorage         *storage.PrefixedAtomicStorage
	providerControlService     *escrow.ProviderControlService
	freeCallStateService       *escrow.FreeCallStateService
	daemonHeartbeat            *metrics.DaemonHeartbeat
	paymentStorage             *escrow.PaymentStorage
	priceStrategy              *pricing.PricingStrategy
	configurationService       *configuration_service.ConfigurationService
	configurationBroadcaster   *configuration_service.MessageBroadcaster
	organizationMetaData       *blockchain.OrganizationMetaData
	prepaidPaymentHandler      handler.PaymentHandler
	prepaidUserStorage         storage.TypedAtomicStorage
	prepaidUserService         escrow.PrePaidService
	freeCallPaymentHandler     handler.PaymentHandler
	freeCallUserService        escrow.FreeCallUserService
	freeCallUserStorage        *escrow.FreeCallUserStorage
	freeCallLockerStorage      *storage.PrefixedAtomicStorage
	tokenManager               token.Manager
	tokenService               *escrow.TokenService
	modelService               training.ModelServer
	modelUserStorage           *training.ModelUserStorage
	modelStorage               *training.ModelStorage
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
	config.Validate()
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

func (components *Components) OrganizationMetaData() *blockchain.OrganizationMetaData {
	if components.organizationMetaData != nil {
		return components.organizationMetaData
	}
	components.organizationMetaData = blockchain.GetOrganizationMetaData()
	return components.organizationMetaData
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

	client, err := etcddb.NewEtcdClient(components.OrganizationMetaData())
	if err != nil {
		log.WithError(err).Panic("unable to create etcd client")
	}

	components.etcdClient = client
	return components.etcdClient
}

func (components *Components) FreeCallLockerStorage() *storage.PrefixedAtomicStorage {
	if components.freeCallLockerStorage != nil {
		return components.freeCallLockerStorage
	}
	components.freeCallLockerStorage = storage.NewPrefixedAtomicStorage(components.AtomicStorage(), "/freecall/lock")
	return components.freeCallLockerStorage
}

func (components *Components) LockerStorage() *storage.PrefixedAtomicStorage {
	if components.etcdLockerStorage != nil {
		return components.etcdLockerStorage
	}
	components.etcdLockerStorage = storage.NewPrefixedAtomicStorage(components.MPESpecificStorage(), "/payment-channel/lock")
	return components.etcdLockerStorage
}

/*
create new PrefixedStorage using /<network_name> as a prefix, use this storage as base for other storages
(i.e. return it from GetAtomicStorage of components.go);
this guarantees that storages for different networks never intersect
*/
func (components *Components) AtomicStorage() storage.AtomicStorage {
	var store storage.AtomicStorage
	if components.atomicStorage != nil {
		return components.atomicStorage
	}

	if config.GetString(config.PaymentChannelStorageTypeKey) == "etcd" {
		store = components.EtcdClient()
	} else {
		store = storage.NewMemStorage()
	}
	//by default set the network selected in the storage path
	components.atomicStorage = storage.NewPrefixedAtomicStorage(store, config.GetString(config.BlockChainNetworkSelected)+"/"+config.GetString(config.OrganizationId)+"/"+components.OrganizationMetaData().GetGroupIdString())

	return components.atomicStorage
}

/*
add new component MPESpecificStorage; it is also instance of PrefixedStorage using /<mpe_contract_address> as a prefix; as it is also based on storage from previous item the effective prefix is /<network_id>/<mpe_contract_address>; this guarantees that storages which are specific for MPE contract version don't intersect;
use MPESpecificStorage as base for PaymentChannelStorage, PaymentStorage, LockStorage for channels;
*/
func (components *Components) MPESpecificStorage() *storage.PrefixedAtomicStorage {
	if components.mpeSpecificStorage != nil {
		return components.mpeSpecificStorage
	}
	components.mpeSpecificStorage = storage.NewPrefixedAtomicStorage(components.AtomicStorage(), components.ServiceMetaData().MpeAddress)
	return components.mpeSpecificStorage
}

func (components *Components) PaymentStorage() *escrow.PaymentStorage {
	if components.paymentStorage != nil {
		return components.paymentStorage
	}

	components.paymentStorage = escrow.NewPaymentStorage(components.MPESpecificStorage())

	return components.paymentStorage
}

func (components *Components) FreeCallUserStorage() *escrow.FreeCallUserStorage {
	if components.freeCallUserStorage != nil {
		return components.freeCallUserStorage
	}

	components.freeCallUserStorage = escrow.NewFreeCallUserStorage(components.AtomicStorage())

	return components.freeCallUserStorage
}

func (components *Components) PrepaidUserStorage() storage.TypedAtomicStorage {
	if components.prepaidUserStorage != nil {
		return components.prepaidUserStorage
	}

	components.prepaidUserStorage = escrow.NewPrepaidStorage(components.AtomicStorage())

	return components.prepaidUserStorage
}
func (components *Components) PaymentChannelService() escrow.PaymentChannelService {
	if components.paymentChannelService != nil {
		return components.paymentChannelService
	}

	components.paymentChannelService = escrow.NewPaymentChannelService(
		escrow.NewPaymentChannelStorage(components.MPESpecificStorage()),
		components.PaymentStorage(),
		escrow.NewBlockchainChannelReader(components.Blockchain(), config.Vip(), components.OrganizationMetaData()),
		escrow.NewEtcdLocker(components.LockerStorage()),
		escrow.NewChannelPaymentValidator(components.Blockchain(), config.Vip(), components.OrganizationMetaData()), func() ([32]byte, error) {
			s := components.OrganizationMetaData().GetGroupId()
			return s, nil
		},
	)

	return components.paymentChannelService
}

func (components *Components) FreeCallUserService() escrow.FreeCallUserService {
	if components.freeCallUserService != nil {
		return components.freeCallUserService
	}

	components.freeCallUserService = escrow.NewFreeCallUserService(
		components.FreeCallUserStorage(),
		escrow.NewEtcdLocker(components.FreeCallLockerStorage()),
		func() ([32]byte, error) {
			s := components.OrganizationMetaData().GetGroupId()
			return s, nil
		}, components.ServiceMetaData())

	return components.freeCallUserService
}

func (components *Components) EscrowPaymentHandler() handler.PaymentHandler {
	if components.escrowPaymentHandler != nil {
		return components.escrowPaymentHandler
	}

	components.escrowPaymentHandler = escrow.NewPaymentHandler(
		components.PaymentChannelService(),
		components.Blockchain(),
		escrow.NewIncomeValidator(components.PricingStrategy()),
	)

	return components.escrowPaymentHandler
}

func (components *Components) FreeCallPaymentHandler() handler.PaymentHandler {
	if components.freeCallPaymentHandler != nil {
		return components.freeCallPaymentHandler
	}

	components.freeCallPaymentHandler = escrow.FreeCallPaymentHandler(components.FreeCallUserService(),
		components.Blockchain(), components.OrganizationMetaData(), components.ServiceMetaData())

	return components.freeCallPaymentHandler
}

func (components *Components) AllowedUserPaymentHandler() handler.PaymentHandler {
	if components.allowedUserPaymentHandler != nil {
		return components.allowedUserPaymentHandler
	}

	components.allowedUserPaymentHandler = escrow.AllowedUserPaymentHandler()

	return components.allowedUserPaymentHandler
}

func (components *Components) PrePaidPaymentHandler() handler.PaymentHandler {
	if components.prepaidPaymentHandler != nil {
		return components.PrePaidPaymentHandler()
	}

	components.prepaidPaymentHandler = escrow.
		NewPrePaidPaymentHandler(components.PrePaidService(), components.OrganizationMetaData(), components.ServiceMetaData(),
			components.PricingStrategy(), components.TokenManager())

	return components.prepaidPaymentHandler
}

func (components *Components) PrePaidService() escrow.PrePaidService {
	if components.prepaidUserService != nil {
		return components.prepaidUserService
	}
	components.prepaidUserService = escrow.NewPrePaidService(components.PrepaidUserStorage(),
		escrow.NewPrePaidPaymentValidator(components.PricingStrategy(), components.TokenManager()), func() ([32]byte, error) {
			s := components.OrganizationMetaData().GetGroupId()
			return s, nil
		})
	return components.prepaidUserService
}

// Add a chain of interceptors
func (components *Components) GrpcInterceptor() grpc.StreamServerInterceptor {
	if components.grpcInterceptor != nil {
		return components.grpcInterceptor
	}
	//Metering is now mandatory in Daemon
	metrics.SetDaemonGrpId(components.OrganizationMetaData().GetGroupIdString())
	if components.Blockchain().Enabled() && config.GetBool(config.MeteringEnabled) {

		meteringUrl := config.GetString(config.MeteringEndPoint) + "/metering/verify"
		if ok, err := components.verifyAuthenticationSetUpForFreeCall(meteringUrl,
			components.OrganizationMetaData().GetGroupIdString()); !ok {
			log.Error(err)
			log.WithError(err).Panic("Metering authentication failed.Please verify the configuration" +
				" as part of service publication process")
		}

		components.grpcInterceptor = grpc_middleware.ChainStreamServer(
			handler.GrpcMeteringInterceptor(), handler.GrpcRateLimitInterceptor(components.ChannelBroadcast()),
			components.GrpcPaymentValidationInterceptor())
	} else {
		components.grpcInterceptor = grpc_middleware.ChainStreamServer(handler.GrpcRateLimitInterceptor(components.ChannelBroadcast()),
			components.GrpcPaymentValidationInterceptor())
	}
	return components.grpcInterceptor
}

// Metering end point authentication is now mandatory for daemon
func (components *Components) verifyAuthenticationSetUpForFreeCall(serviceURL string, groupId string) (ok bool, err error) {

	if _, err = crypto.HexToECDSA(config.GetString(config.PvtKeyForMetering)); err != nil {
		return false, errors.New("you need a specify a valid private key 'pvt_key_for_metering' as part of service publication process." + err.Error())
	}

	req, err := http.NewRequest("GET", serviceURL, nil)
	if err != nil {
		log.WithField("serviceURL", serviceURL).WithError(err).Warningf("Unable to create service request to publish stats")
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-authtype", "verification")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	metrics.SignMessageForMetering(req,
		&metrics.CommonStats{OrganizationID: config.GetString(config.OrganizationId), ServiceID: config.GetString(config.ServiceId),
			GroupID: groupId, UserName: metrics.GetDaemonID()})

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return checkResponse(response)

}

// Check if the response received was proper
func checkResponse(response *http.Response) (allowed bool, err error) {
	if response == nil {
		log.Error("Empty response received.")
		return false, fmt.Errorf("Empty response received.")
	}
	if response.StatusCode != http.StatusOK {
		log.Errorf("Service call failed with status code : %d ", response.StatusCode)
		return false, fmt.Errorf("Service call failed with status code : %d ", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Infof("Unable to retrieve calls allowed from Body , : %s ", err.Error())
		return false, err
	}
	var responseBody VerifyMeteringResponse
	if err = json.Unmarshal(body, &responseBody); err != nil {
		return false, err
	}
	//close the body
	defer response.Body.Close()

	if strings.Compare(responseBody.Data, "success") != 0 {
		return false, fmt.Errorf("Error returned by by Metering Service %s Verification,"+
			" pls check the pvt_key_for_metering set up. The public key in metering does not correspond "+
			"to the private key in Daemon config.", config.GetString(config.MeteringEndPoint)+"/verify")
	}

	return true, nil
}

type VerifyMeteringResponse struct {
	Data string `json:"data"`
}

func (components *Components) GrpcPaymentValidationInterceptor() grpc.StreamServerInterceptor {
	if !components.Blockchain().Enabled() {
		if config.GetBool(config.AllowedUserFlag) {
			log.Info("Blockchain is disabled And AllowedUserFlag is enabled")
			return handler.GrpcPaymentValidationInterceptor(components.ServiceMetaData(), components.AllowedUserPaymentHandler())

		}
		log.Info("Blockchain is disabled: no payment validation")
		return handler.NoOpInterceptor
	} else {
		log.Info("Blockchain is enabled: instantiate payment validation interceptor")
		return handler.GrpcPaymentValidationInterceptor(components.ServiceMetaData(), components.EscrowPaymentHandler(),
			components.FreeCallPaymentHandler(), components.PrePaidPaymentHandler())
	}
}

func (components *Components) PaymentChannelStateService() (service escrow.PaymentChannelStateServiceServer) {
	if !config.GetBool(config.BlockchainEnabledKey) {
		return &escrow.BlockChainDisabledStateService{}
	}

	if components.paymentChannelStateService != nil {
		return components.paymentChannelStateService
	}

	components.paymentChannelStateService = escrow.NewPaymentChannelStateService(
		components.PaymentChannelService(),
		components.PaymentStorage(),
		components.ServiceMetaData())

	return components.paymentChannelStateService
}

//NewProviderControlService

func (components *Components) ProviderControlService() (service escrow.ProviderControlServiceServer) {

	if !config.GetBool(config.BlockchainEnabledKey) {
		return &escrow.BlockChainDisabledProviderControlService{}
	}
	if components.providerControlService != nil {
		return components.providerControlService
	}

	components.providerControlService = escrow.NewProviderControlService(components.PaymentChannelService(),
		components.ServiceMetaData(), components.OrganizationMetaData())
	return components.providerControlService
}

func (components *Components) FreeCallStateService() (service escrow.FreeCallStateServiceServer) {

	if !config.GetBool(config.BlockchainEnabledKey) {
		return &escrow.BlockChainDisabledFreeCallStateService{}
	}
	if components.freeCallStateService != nil {
		return components.freeCallStateService
	}

	components.freeCallStateService = escrow.NewFreeCallStateService(components.OrganizationMetaData(),
		components.ServiceMetaData(), components.FreeCallUserService(),
		escrow.NewFreeCallPaymentValidator(components.Blockchain().CurrentBlock,
			components.ServiceMetaData().FreeCallSignerAddress()))
	return components.freeCallStateService
}

func (components *Components) DaemonHeartBeat() (service *metrics.DaemonHeartbeat) {
	if components.daemonHeartbeat != nil {
		return components.daemonHeartbeat
	}
	metrics.SetDaemonGrpId(components.OrganizationMetaData().GetGroupIdString())
	components.daemonHeartbeat = &metrics.DaemonHeartbeat{DaemonID: metrics.GetDaemonID(), DaemonVersion: config.GetVersionTag()}
	return components.daemonHeartbeat
}

func (components *Components) PricingStrategy() *pricing.PricingStrategy {
	if components.priceStrategy != nil {
		return components.priceStrategy
	}

	components.priceStrategy, _ = pricing.InitPricingStrategy(components.ServiceMetaData())

	return components.priceStrategy
}

func (components *Components) ChannelBroadcast() *configuration_service.MessageBroadcaster {
	if components.configurationBroadcaster != nil {
		return components.configurationBroadcaster
	}

	components.configurationBroadcaster = configuration_service.NewChannelBroadcaster()

	return components.configurationBroadcaster
}

func (components *Components) ConfigurationService() *configuration_service.ConfigurationService {
	if components.configurationService != nil {
		return components.configurationService
	}

	components.configurationService = configuration_service.NewConfigurationService(components.ChannelBroadcast())

	return components.configurationService
}

func (components *Components) ModelUserStorage() *training.ModelUserStorage {
	if components.modelUserStorage != nil {
		return components.modelUserStorage
	}
	components.modelUserStorage = training.NewUerModelStorage(components.AtomicStorage())
	return components.modelUserStorage
}

func (components *Components) ModelStorage() *training.ModelStorage {
	if components.modelStorage != nil {
		return components.modelStorage
	}
	components.modelStorage = training.NewModelStorage(components.AtomicStorage())
	return components.modelStorage
}

func (components *Components) ModelService() training.ModelServer {
	if components.modelService != nil {
		return components.modelService
	}
	if !config.GetBool(config.BlockchainEnabledKey) {
		components.modelService = &training.NoModelSupportService{}
		return components.modelService
	}

	components.modelService = training.NewModelService(components.PaymentChannelService(), components.ServiceMetaData(),
		components.OrganizationMetaData(), components.ModelStorage(), components.ModelUserStorage())
	return components.modelService
}

func (components *Components) TokenManager() token.Manager {
	if components.tokenManager != nil {
		return components.tokenManager
	}

	components.tokenManager = token.NewJWTTokenService(*components.OrganizationMetaData())

	return components.tokenManager
}

func (components *Components) TokenService() escrow.TokenServiceServer {
	if components.tokenService != nil {
		return components.tokenService
	}
	if !config.GetBool(config.BlockchainEnabledKey) {
		return &escrow.BlockChainDisabledTokenService{}
	}

	components.tokenService = escrow.NewTokenService(components.PaymentChannelService(),
		components.PrePaidService(), components.TokenManager(),
		escrow.NewChannelPaymentValidator(components.Blockchain(), config.Vip(), components.OrganizationMetaData()),
		components.ServiceMetaData())

	return components.tokenService
}
