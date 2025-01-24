package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/training"
)

type DaemonServiceSuite struct {
	suite.Suite
	modelStorage               *training.ModelStorage
	userModelStorage           *training.ModelUserStorage
	pendingModelStorage        *training.PendingModelStorage
	daemonService              training.DaemonServer
	unimplementedDaemonService training.DaemonServer
	modelKeys                  []*training.ModelKey
	pendingModelKeys           []*training.ModelKey // using for checking updated status
	grpcServer                 *grpc.Server
	grpcClient                 *grpc.ClientConn
	serviceMetadata            *blockchain.ServiceMetadata
	organizationMetadata       *blockchain.OrganizationMetaData
}

func TestDaemonServiceSuite(t *testing.T) {
	suite.Run(t, new(DaemonServiceSuite))
}

var (
	testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"test_org_id\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",  \"license_server_endpoints\": [\"https://licensendpoint:8082\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonServiceData  = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "
	testUserAddress = "0x3432cBa6BF635Df5fBFD1f1a794fA66D412b8774"
)

func (suite *DaemonServiceSuite) SetupSuite() {
	// setup test config once
	suite.setupTestConfig()

	// init service metadata and organization metadata
	serviceMetadata, err := blockchain.InitServiceMetaDataFromJson([]byte(testJsonServiceData))
	orgMetadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	if err != nil {
		zap.L().Fatal("Error in initinalize organization metadata from json", zap.Error(err))
	}

	suite.serviceMetadata = serviceMetadata
	suite.organizationMetadata = orgMetadata

	// setup unimplemented daemon server once
	suite.unimplementedDaemonService = training.NoTrainingDaemonServer{}

	// setup test poriver service once
	address := "localhost:5001"
	suite.grpcServer = startTestService(address)
}

func (suite *DaemonServiceSuite) SetupTest() {
	// setup storages before each test for isolation environment
	modelStorage, userModelStorage, pendingModelStorage, publicModelStorage := suite.createTestModels()
	suite.modelStorage = modelStorage
	suite.userModelStorage = userModelStorage
	suite.pendingModelStorage = pendingModelStorage

	suite.daemonService = training.NewTrainingService(
		nil,
		suite.serviceMetadata,
		suite.organizationMetadata,
		modelStorage,
		userModelStorage,
		pendingModelStorage,
		publicModelStorage,
	)
}

func (suite *DaemonServiceSuite) TearDownSuite() {
	suite.grpcServer.Stop()
}

func getTestSignature(text string, blockNumber int, privateKey *ecdsa.PrivateKey) (signature []byte) {
	HashPrefix32Bytes := []byte("\x19Ethereum Signed Message:\n32")

	message := bytes.Join([][]byte{
		[]byte(text),
		crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
		math.U256Bytes(big.NewInt(int64(blockNumber))),
	}, nil)

	hash := crypto.Keccak256(
		HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		zap.L().Fatal("Cannot sign test message", zap.Error(err))
	}

	return signature
}

func createTestAuthDetails() *training.AuthorizationDetails {
	privateKey, err := crypto.HexToECDSA("c0e4803a3a5b3c26cfc96d19a6dc4bbb4ba653ce5fa68f0b7dbf3903cda17ee6")
	if err != nil {
		zap.L().Fatal("error in creating private key", zap.Error(err))
	}
	return &training.AuthorizationDetails{
		CurrentBlock:  0,
		Message:       "__CreateModel",
		Signature:     getTestSignature("__CreateModel", 0, privateKey),
		SignerAddress: "0x3432cBa6BF635Df5fBFD1f1a794fA66D412b8774",
	}
}

func creatBadTestAuthDetails() *training.AuthorizationDetails {
	privateKey, err := crypto.HexToECDSA("c0e4803a3a5b3c26cfc96d19a6dc4bbb4ba653ce5fa68f0b7dbf3903cda17ee6")
	if err != nil {
		zap.L().Fatal("error in creating private key", zap.Error(err))
	}
	return &training.AuthorizationDetails{
		CurrentBlock:  0,
		Message:       "badMessage",
		Signature:     getTestSignature("badMessage", 0, privateKey),
		SignerAddress: "0x4444cBa6BF635Df5fBFD1f1a794fA66D412b8774",
	}
}

func (suite *DaemonServiceSuite) setupTestConfig() {
	testConfigJson := `
{
	"blockchain_enabled": true,
	"blockchain_network_selected": "sepolia",
	"daemon_end_point": "127.0.0.1:8080",
	"daemon_group_name":"default_group",
	"payment_channel_storage_type": "etcd",
	"ipfs_end_point": "http://ipfs.singularitynet.io:80",
	"ipfs_timeout" : 30,
	"passthrough_enabled": true,
	"passthrough_endpoint":"http://127.0.0.1:5002",
	"service_id": "service_id",
	"organization_id": "test_org_id",
	"metering_enabled": false,
	"ssl_cert": "",
	"ssl_key": "",
	"max_message_size_in_mb" : 4,
	"daemon_type": "grpc",
    "enable_dynamic_pricing":false,
	"allowed_user_flag" :false,
	"auto_ssl_domain": "",
	"auto_ssl_cache_dir": ".certs",
	"private_key": "",
	"log":  {
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "text",
			"timestamp_format": "2006-01-02T15:04:05.999Z07:00"
		},
		"output": {
			"type": ["file", "stdout"],
			"file_pattern": "./snet-daemon.%Y%m%d.log",
			"current_link": "./snet-daemon.log",
			"max_size_in_mb": 10,
			"max_age_in_days": 7,
			"rotation_count": 0
		},
		"hooks": []
	},
	"model_maintenance_endpoint": "http://localhost:5001",
	"payment_channel_storage_client": {
		"connection_timeout": "0s",
		"request_timeout": "0s",
		"hot_reload": true
    },
	"payment_channel_storage_server": {
		"id": "storage-1",
		"scheme": "http",
		"host" : "127.0.0.1",
		"client_port": 2379,
		"peer_port": 2380,
		"token": "unique-token",
		"cluster": "storage-1=http://127.0.0.1:2380",
		"startup_timeout": "1m",
		"data_dir": "storage-data-dir-1.etcd",
		"log_level": "info",
		"log_outputs": ["./etcd-server.log"],
		"enabled": false
	},
	"alerts_email": "", 
	"service_heartbeat_type": "http",
    "token_expiry_in_minutes": 1440,
    "model_training_enabled": false
}`

	var testConfig = viper.New()
	err := config.ReadConfigFromJsonString(testConfig, testConfigJson)
	if err != nil {
		zap.L().Fatal("Error in reading config")
	}

	config.SetVip(testConfig)
}

func (suite *DaemonServiceSuite) createTestModels() (*training.ModelStorage, *training.ModelUserStorage, *training.PendingModelStorage, *training.PublicModelStorage) {
	memStorage := storage.NewMemStorage()
	modelStorage := training.NewModelStorage(memStorage)
	userModelStorage := training.NewUserModelStorage(memStorage)
	pendingModelStorage := training.NewPendingModelStorage(memStorage)
	publicModelStorage := training.NewPublicModelStorage(memStorage)

	modelA := &training.ModelData{
		IsPublic:            true,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              training.Status_VALIDATING,
		CreatedByAddress:    "address",
		ModelId:             "test_1",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		IsDefault:           true,
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelAKey := &training.ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_1",
	}

	modelB := &training.ModelData{
		IsPublic:            false,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              training.Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "test_2",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		IsDefault:           true,
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelBKey := &training.ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_2",
	}

	modelC := &training.ModelData{
		IsPublic:            false,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              training.Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "test_3",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		IsDefault:           true,
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelCKey := &training.ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_3",
	}

	modelStorage.Put(modelAKey, modelA)
	modelStorage.Put(modelBKey, modelB)
	modelStorage.Put(modelCKey, modelC)

	// adding to user models sotrage
	userModelKey := &training.ModelUserKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    testUserAddress,
	}

	userModelData := &training.ModelUserData{
		ModelIds:       []string{"test_3"},
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    testUserAddress,
	}

	userModelStorage.Put(userModelKey, userModelData)

	// adding to pending models storage
	pendingModelKey := &training.PendingModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	pendingModelsData := &training.PendingModelData{
		ModelIDs: []string{"test_1"},
	}

	pendingModelStorage.Put(pendingModelKey, pendingModelsData)

	// adding to public models storage
	publicModelKey := &training.PublicModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	publicModelsData := &training.PublicModelData{
		ModelIDs: []string{"test_1"},
	}

	publicModelStorage.Put(publicModelKey, publicModelsData)

	// setup keys in suite
	suite.modelKeys = []*training.ModelKey{modelAKey, modelBKey, modelCKey}
	suite.pendingModelKeys = []*training.ModelKey{modelAKey}

	// return all model keys, storages
	return modelStorage, userModelStorage, pendingModelStorage, publicModelStorage
}

func (suite *DaemonServiceSuite) createAdditionalTestModel(modelName string, authDetails *training.AuthorizationDetails) string {
	newModel := &training.NewModel{
		Name:            modelName,
		Description:     "test_desc",
		GrpcMethodName:  "test_grpc_method_name",
		GrpcServiceName: "test_grpc_service_name",
		AddressList:     []string{},
		IsPublic:        false,
		OrganizationId:  "test_org_id",
		ServiceId:       "service_id",
		GroupId:         "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	request := &training.NewModelRequest{
		Authorization: authDetails,
		Model:         newModel,
	}
	response, err := suite.daemonService.CreateModel(context.Background(), request)
	if err != nil {
		zap.L().Fatal("error in creating additional test model", zap.Error(err))
	}

	return response.ModelId
}

func (suite *DaemonServiceSuite) TestDaemonService_GetModel() {
	testAuthCreads := createTestAuthDetails()
	badTestAuthCreads := creatBadTestAuthDetails()

	// check without request
	response1, err := suite.daemonService.GetModel(context.Background(), nil)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response1.Status)

	// check without auth
	request2 := &training.CommonRequest{
		Authorization: nil,
		ModelId:       "test_2",
	}
	response2, err := suite.daemonService.GetModel(context.Background(), request2)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response2.Status)

	// check with bad auth
	request3 := &training.CommonRequest{
		Authorization: badTestAuthCreads,
		ModelId:       "test_2",
	}
	response3, err := suite.daemonService.GetModel(context.Background(), request3)
	assert.ErrorContains(suite.T(), err, training.ErrBadAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response3.Status)

	// check modelId is not empty string
	request4 := &training.CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "",
	}
	response4, err := suite.daemonService.GetModel(context.Background(), request4)
	assert.NotNil(suite.T(), err)
	assert.ErrorContains(suite.T(), err, training.ErrEmptyModelID.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response4.Status)

	// check without access to model
	request5 := &training.CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_2",
	}
	response5, err := suite.daemonService.GetModel(context.Background(), request5)
	assert.ErrorContains(suite.T(), err, training.ErrAccessToModel.Error())
	assert.Equal(suite.T(), &training.ModelResponse{}, response5)

	// check access to public model
	request6 := &training.CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_1",
	}
	response6, err := suite.daemonService.GetModel(context.Background(), request6)
	assert.Nil(suite.T(), err)
	assert.NotEmpty(suite.T(), response6)
	assert.Equal(suite.T(), true, response6.IsPublic)

	//check access to non public model
	request7 := &training.CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_3",
	}
	response7, err := suite.daemonService.GetModel(context.Background(), request7)
	assert.Nil(suite.T(), err)
	assert.NotEmpty(suite.T(), response7)
	assert.Equal(suite.T(), false, response7.IsPublic)
}

func (suite *DaemonServiceSuite) TestDaemonService_CreateModel() {
	testAuthCreads := createTestAuthDetails()
	badTestAuthCreads := creatBadTestAuthDetails()

	newModel := &training.NewModel{
		Name:            "new_test_model",
		Description:     "test_desc",
		GrpcMethodName:  "test_grpc_method_name",
		GrpcServiceName: "test_grpc_service_name",
		AddressList:     []string{},
		IsPublic:        false,
		OrganizationId:  "test_org_id",
		ServiceId:       "service_id",
		GroupId:         "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	newEmptyModel := &training.NewModel{}

	// check without request
	response1, err := suite.daemonService.CreateModel(context.Background(), nil)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response1.Status)

	// check without auth
	request2 := &training.NewModelRequest{
		Authorization: nil,
		Model:         newModel,
	}
	response2, err := suite.daemonService.CreateModel(context.Background(), request2)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response2.Status)

	// check with bad auth
	request3 := &training.NewModelRequest{
		Authorization: badTestAuthCreads,
		Model:         newModel,
	}
	response3, err := suite.daemonService.CreateModel(context.Background(), request3)
	assert.ErrorContains(suite.T(), err, training.ErrBadAuthorization.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response3.Status)

	// check with emptyModel
	request4 := &training.NewModelRequest{
		Authorization: testAuthCreads,
		Model:         newEmptyModel,
	}
	response4, err := suite.daemonService.CreateModel(context.Background(), request4)
	assert.ErrorContains(suite.T(), err, training.ErrNoGRPCServiceOrMethod.Error())
	assert.Equal(suite.T(), training.Status_ERRORED, response4.Status)

	// check with auth
	request5 := &training.NewModelRequest{
		Authorization: testAuthCreads,
		Model:         newModel,
	}
	response5, err := suite.daemonService.CreateModel(context.Background(), request5)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_CREATED, response5.Status)

	// check model creation in model storage
	modelKey := &training.ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        response5.ModelId,
	}

	modelData, ok, err := suite.modelStorage.Get(modelKey)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), response5.ModelId, modelData.ModelId)
	assert.Equal(suite.T(), newModel.Name, modelData.ModelName)

	// check user model data creation in user model storage
	userModelKey := &training.ModelUserKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    testUserAddress,
	}

	userModelData, ok, err := suite.userModelStorage.Get(userModelKey)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), ok)
	assert.True(suite.T(), slices.Contains(userModelData.ModelIds, response5.ModelId))
}

func (suite *DaemonServiceSuite) TestDaemonService_GetAllModels() {
	testAuthCreads := createTestAuthDetails()
	badTestAuthCreads := creatBadTestAuthDetails()

	newAdditionalTestModelId := suite.createAdditionalTestModel("new_additional_test_model", testAuthCreads)

	expectedModelIds := []string{"test_3", newAdditionalTestModelId, "test_1"}

	// check without request
	response1, err := suite.daemonService.GetAllModels(context.Background(), nil)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Nil(suite.T(), response1.ListOfModels)

	// check without auth
	request2 := &training.AllModelsRequest{
		Authorization: nil,
	}
	response2, err := suite.daemonService.GetAllModels(context.Background(), request2)
	assert.ErrorContains(suite.T(), err, training.ErrNoAuthorization.Error())
	assert.Nil(suite.T(), response2.ListOfModels)

	// check with bad auth
	request3 := &training.AllModelsRequest{
		Authorization: badTestAuthCreads,
	}
	response3, err := suite.daemonService.GetAllModels(context.Background(), request3)
	assert.ErrorContains(suite.T(), err, training.ErrBadAuthorization.Error())
	assert.Nil(suite.T(), response3.ListOfModels)

	// check with auth and without filters
	request4 := &training.AllModelsRequest{
		Authorization: testAuthCreads,
	}
	response4, err := suite.daemonService.GetAllModels(context.Background(), request4)
	assert.Nil(suite.T(), err)
	modelIds := []string{}
	for _, model := range response4.ListOfModels {
		modelIds = append(modelIds, model.ModelId)
	}

	assert.True(suite.T(), slices.Equal(expectedModelIds, modelIds))
}

func (suite *DaemonServiceSuite) TestDaemonSerice_ManageUpdateStatusWorkers() {
	duration := time.Second * 10
	deadline := time.Now().Add(duration)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		zap.L().Info("Context done", zap.Error(ctx.Err()))
	case <-time.After(duration):
		zap.L().Info("Operation timed out after", zap.Duration("duration", duration))
	}

	for _, modelKey := range suite.pendingModelKeys {
		modelData, ok, err := suite.modelStorage.Get(modelKey)
		assert.Nil(suite.T(), err)
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), training.Status_VALIDATED, modelData.Status)
	}
}

func (suite *DaemonServiceSuite) TestDaemonService_UnimplementedDaemonService() {
	response1, err := suite.unimplementedDaemonService.CreateModel(context.TODO(), &training.NewModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response1.Status)

	_, err = suite.unimplementedDaemonService.ValidateModelPrice(context.TODO(), &training.AuthValidateRequest{})
	assert.NotNil(suite.T(), err)

	response2, err := suite.unimplementedDaemonService.ValidateModel(context.TODO(), &training.AuthValidateRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response2.Status)

	_, err = suite.unimplementedDaemonService.TrainModelPrice(context.TODO(), &training.CommonRequest{})
	assert.NotNil(suite.T(), err)

	response3, err := suite.unimplementedDaemonService.TrainModel(context.TODO(), &training.CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response3.Status)

	response4, err := suite.unimplementedDaemonService.DeleteModel(context.TODO(), &training.CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response4.Status)

	response5, err := suite.unimplementedDaemonService.GetAllModels(context.TODO(), &training.AllModelsRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), []*training.ModelResponse{}, response5.ListOfModels)

	response6, err := suite.unimplementedDaemonService.GetModel(context.TODO(), &training.CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response6.Status)

	response7, err := suite.unimplementedDaemonService.UpdateModel(context.Background(), &training.UpdateModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), training.Status_ERRORED, response7.Status)

	_, err = suite.unimplementedDaemonService.GetTrainingMetadata(context.Background(), &emptypb.Empty{})
	assert.NotNil(suite.T(), err)

	_, err = suite.unimplementedDaemonService.GetMethodMetadata(context.TODO(), &training.MethodMetadataRequest{})
	assert.NotNil(suite.T(), err)
}
