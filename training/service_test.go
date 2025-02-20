package training

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/storage"
)

type DaemonServiceSuite struct {
	suite.Suite
	blockchain                 blockchain.Processor
	currentBlock               *big.Int
	modelStorage               *ModelStorage
	userModelStorage           *ModelUserStorage
	pendingModelStorage        *PendingModelStorage
	daemonService              DaemonServer
	unimplementedDaemonService DaemonServer
	modelKeys                  []*ModelKey
	pendingModelKeys           []*ModelKey // using for checking updated status
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
	err := config.Validate()
	assert.Nil(suite.T(), err)

	// init service metadata and organization metadata
	serviceMetadata, err := blockchain.InitServiceMetaDataFromJson([]byte(testJsonServiceData))
	suite.blockchain, err = blockchain.NewProcessor(serviceMetadata)
	if err != nil {
		suite.T().Fatalf("can't connect to blockchain: %v", err)
	}
	suite.currentBlock, err = suite.blockchain.CurrentBlock()

	orgMetadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	if err != nil {
		suite.T().Fatalf("Error in initinalize organization metadata from json: %v", err)
	}

	suite.serviceMetadata = serviceMetadata
	suite.organizationMetadata = orgMetadata

	// setup unimplemented daemon server once
	suite.unimplementedDaemonService = NoTrainingDaemonServer{}

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

	suite.daemonService = NewTrainingService(
		&suite.blockchain,
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

func getTestSignature(text string, blockNumber uint64, privateKey *ecdsa.PrivateKey) (signature []byte) {
	HashPrefix32Bytes := []byte("\x19Ethereum Signed Message:\n32")

	message := bytes.Join([][]byte{
		[]byte(text),
		crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
		math.U256Bytes(big.NewInt(0).SetUint64(blockNumber)),
	}, nil)

	hash := crypto.Keccak256(
		HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil
	}

	return signature
}

func createTestAuthDetails(block *big.Int, method string) *AuthorizationDetails {
	privateKey, err := crypto.HexToECDSA("c0e4803a3a5b3c26cfc96d19a6dc4bbb4ba653ce5fa68f0b7dbf3903cda17ee6")
	if err != nil {
		return nil
	}
	return &AuthorizationDetails{
		CurrentBlock:  block.Uint64(),
		Message:       method,
		Signature:     getTestSignature(method, block.Uint64(), privateKey),
		SignerAddress: "0x3432cBa6BF635Df5fBFD1f1a794fA66D412b8774",
	}
}

func creatBadTestAuthDetails(block *big.Int) *AuthorizationDetails {
	privateKey, err := crypto.HexToECDSA("c0e4803a3a5b3c26cfc96d19a6dc4bbb4ba653ce5fa68f0b7dbf3903cda17ee6")
	if err != nil {
		return nil
	}
	return &AuthorizationDetails{
		CurrentBlock:  block.Uint64(),
		Message:       "badMessage",
		Signature:     getTestSignature("badMessage", block.Uint64(), privateKey),
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
	"payment_channel_storage_type": "_etcd",
	"ipfs_end_point": "http://ipfs.singularitynet.io:80",
  	"ethereum_json_rpc_http_endpoint": "https://sepolia.infura.io/v3/09027f4a13e841d48dbfefc67e7685d5",
	"ipfs_timeout" : 30,
	"passthrough_enabled": true,
	"passthrough_endpoint":"http://0.0.0.0:5001",
	"model_maintenance_endpoint": "http://0.0.0.0:5001",
	"model_training_endpoint": "http://0.0.0.0:5001",
	"service_id": "service_id",
	"organization_id": "test_org_id",
	"metering_enabled": false,
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
			"type": ["stdout"]
		},
		"hooks": []
	},
	"payment_channel_storage_client": {
		"connection_timeout": "0s",
		"request_timeout": "0s",
		"hot_reload": true
    },
	"payment_channel_storage_server": {
		"enabled": false
	},
	"alerts_email": "", 
	"service_heartbeat_type": "http",
    "model_training_enabled": true
}`

	var testConfig = viper.New()
	err := config.ReadConfigFromJsonString(testConfig, testConfigJson)
	if err != nil {
		suite.T().Fatalf("Error in reading config")
	}

	config.SetVip(testConfig)
}

func (suite *DaemonServiceSuite) createTestModels() (*ModelStorage, *ModelUserStorage, *PendingModelStorage, *PublicModelStorage) {
	memStorage := storage.NewMemStorage()
	modelStorage := NewModelStorage(memStorage, suite.organizationMetadata)
	userModelStorage := NewUserModelStorage(memStorage, suite.organizationMetadata)
	pendingModelStorage := NewPendingModelStorage(memStorage, suite.organizationMetadata)
	publicModelStorage := NewPublicModelStorage(memStorage, suite.organizationMetadata)

	modelA := &ModelData{
		IsPublic:            true,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              Status_VALIDATING,
		CreatedByAddress:    "address",
		ModelId:             "test_1",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelAKey := &ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_1",
	}

	modelB := &ModelData{
		IsPublic:            false,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "test_2",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelBKey := &ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_2",
	}

	modelC := &ModelData{
		IsPublic:            false,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "test_3",
		UpdatedByAddress:    "string",
		GroupId:             "string",
		OrganizationId:      "string",
		ServiceId:           "string",
		GRPCMethodName:      "string",
		GRPCServiceName:     "string",
		Description:         "string",
		TrainingLink:        "string",
		UpdatedDate:         "string",
	}

	modelCKey := &ModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		ModelId:        "test_3",
	}

	err := modelStorage.Put(modelAKey, modelA)
	if err != nil {
		suite.T().Fatalf("error in putting model: %v", err)
	}
	err = modelStorage.Put(modelBKey, modelB)
	if err != nil {
		suite.T().Fatalf("error in putting model: %v", err)
	}
	err = modelStorage.Put(modelCKey, modelC)
	if err != nil {
		suite.T().Fatalf("error in putting model: %v", err)
	}

	// adding to user models sotrage
	userModelKey := &ModelUserKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    testUserAddress,
	}

	userModelData := &ModelUserData{
		ModelIds:       []string{"test_1", "test_2", "test_3"},
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    testUserAddress,
	}

	err = userModelStorage.Put(userModelKey, userModelData)
	assert.Nil(suite.T(), err)

	// adding to pending models storage
	pendingModelKey := &PendingModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	pendingModelsData := &PendingModelData{
		ModelIDs: []string{"test_1"},
	}

	err = pendingModelStorage.Put(pendingModelKey, pendingModelsData)
	assert.Nil(suite.T(), err)

	// adding to public models storage
	publicModelKey := &PublicModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	publicModelsData := &PublicModelData{
		ModelIDs: []string{"test_1"},
	}

	err = publicModelStorage.Put(publicModelKey, publicModelsData)
	assert.Nil(suite.T(), err)

	// setup keys in suite
	suite.modelKeys = []*ModelKey{modelAKey, modelBKey, modelCKey}
	suite.pendingModelKeys = []*ModelKey{modelAKey}

	// return all model keys, storages
	return modelStorage, userModelStorage, pendingModelStorage, publicModelStorage
}

func (suite *DaemonServiceSuite) createAdditionalTestModel(modelName string, authDetails *AuthorizationDetails) string {
	newModel := &NewModel{
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

	request := &NewModelRequest{
		Authorization: authDetails,
		Model:         newModel,
	}
	response, err := suite.daemonService.CreateModel(context.WithValue(context.Background(), "method", "create_model"), request)
	if err != nil {
		suite.T().Fatalf("error in creating additional test model: %v", err)
	}

	return response.ModelId
}

func (suite *DaemonServiceSuite) TestDaemonService_GetModel() {
	testAuthCreads := createTestAuthDetails(suite.currentBlock, "get_model")
	badTestAuthCreads := creatBadTestAuthDetails(suite.currentBlock)

	// check without request
	response1, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), nil)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response1.Status)

	// check without auth
	request2 := &CommonRequest{
		Authorization: nil,
		ModelId:       "test_2",
	}
	response2, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request2)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response2.Status)

	// check with bad auth
	request3 := &CommonRequest{
		Authorization: badTestAuthCreads,
		ModelId:       "test_2",
	}
	response3, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request3)
	assert.ErrorContains(suite.T(), err, ErrBadAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response3.Status)

	// check modelId is not empty string
	request4 := &CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "",
	}
	response4, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request4)
	assert.NotNil(suite.T(), err)
	assert.ErrorContains(suite.T(), err, ErrEmptyModelID.Error())
	assert.Equal(suite.T(), Status_ERRORED, response4.Status)

	// check without access to model
	request5 := &CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_2",
	}
	response5, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request5)
	assert.ErrorContains(suite.T(), err, ErrAccessToModel.Error())
	assert.Equal(suite.T(), &ModelResponse{}, response5)

	// check access to public model
	request6 := &CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_1",
	}
	response6, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request6)
	assert.Nil(suite.T(), err)
	assert.NotEmpty(suite.T(), response6)
	assert.Equal(suite.T(), true, response6.IsPublic)

	//check access to non public model
	request7 := &CommonRequest{
		Authorization: testAuthCreads,
		ModelId:       "test_3",
	}
	response7, err := suite.daemonService.GetModel(context.WithValue(context.Background(), "method", "get_model"), request7)
	assert.Nil(suite.T(), err)
	assert.NotEmpty(suite.T(), response7)
	assert.Equal(suite.T(), false, response7.IsPublic)
}

func (suite *DaemonServiceSuite) TestDaemonService_CreateModel() {
	var err error
	suite.currentBlock, err = suite.blockchain.CurrentBlock()
	assert.Nil(suite.T(), err)
	testAuthCreads := createTestAuthDetails(suite.currentBlock, "create_model")
	badTestAuthCreads := creatBadTestAuthDetails(suite.currentBlock)

	newModel := &NewModel{
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

	newEmptyModel := &NewModel{}

	// check without request
	response1, err := suite.daemonService.CreateModel(context.WithValue(context.Background(), "method", "create_model"), nil)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response1.Status)

	// check without auth
	request2 := &NewModelRequest{
		Authorization: nil,
		Model:         newModel,
	}
	response2, err := suite.daemonService.CreateModel(context.Background(), request2)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response2.Status)

	// check with bad auth
	request3 := &NewModelRequest{
		Authorization: badTestAuthCreads,
		Model:         newModel,
	}
	response3, err := suite.daemonService.CreateModel(context.WithValue(context.Background(), "method", "create_model"), request3)
	assert.ErrorContains(suite.T(), err, ErrBadAuthorization.Error())
	assert.Equal(suite.T(), Status_ERRORED, response3.Status)
	assert.Nil(suite.T(), err)

	testAuthCreads = createTestAuthDetails(suite.currentBlock, "create_model")

	// check with emptyModel
	request4 := &NewModelRequest{
		Authorization: testAuthCreads,
		Model:         newEmptyModel,
	}
	response4, err := suite.daemonService.CreateModel(context.WithValue(context.Background(), "method", "create_model"), request4)
	assert.ErrorContains(suite.T(), err, ErrNoGRPCServiceOrMethod.Error())
	assert.Equal(suite.T(), Status_ERRORED, response4.Status)

	// check with auth
	request5 := &NewModelRequest{
		Authorization: testAuthCreads,
		Model:         newModel,
	}
	response5, err := suite.daemonService.CreateModel(context.WithValue(context.Background(), "method", "create_model"), request5)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), Status_CREATED, response5.Status)

	// check model creation in model storage
	modelKey := &ModelKey{
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
	userModelKey := &ModelUserKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
		UserAddress:    strings.ToLower(testUserAddress),
	}

	userModelData, ok, err := suite.userModelStorage.Get(userModelKey)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), ok)
	assert.True(suite.T(), slices.Contains(userModelData.ModelIds, response5.ModelId))
}

func (suite *DaemonServiceSuite) TestDaemonService_GetAllModels() {
	testAuthCreads := createTestAuthDetails(suite.currentBlock, "unified")
	testAuthCreadsCreateModel := createTestAuthDetails(suite.currentBlock, "create_model")
	badTestAuthCreads := creatBadTestAuthDetails(suite.currentBlock)

	newAdditionalTestModelId := suite.createAdditionalTestModel("new_additional_test_model", testAuthCreadsCreateModel)

	expectedModelIds := []string{"test_3", newAdditionalTestModelId, "test_1"}

	// check without request
	response1, err := suite.daemonService.GetAllModels(context.WithValue(context.Background(), "method", "get_all_models"), nil)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Nil(suite.T(), response1.ListOfModels)

	// check without auth
	request2 := &AllModelsRequest{
		Authorization: nil,
	}
	response2, err := suite.daemonService.GetAllModels(context.WithValue(context.Background(), "method", "get_all_models"), request2)
	assert.ErrorContains(suite.T(), err, ErrNoAuthorization.Error())
	assert.Nil(suite.T(), response2.ListOfModels)

	// check with bad auth
	request3 := &AllModelsRequest{
		Authorization: badTestAuthCreads,
	}
	response3, err := suite.daemonService.GetAllModels(context.WithValue(context.Background(), "method", "get_all_models"), request3)
	assert.ErrorContains(suite.T(), err, ErrBadAuthorization.Error())
	assert.Nil(suite.T(), response3.ListOfModels)

	// check with auth and without filters
	request4 := &AllModelsRequest{
		Authorization: testAuthCreads,
	}
	response4, err := suite.daemonService.GetAllModels(context.WithValue(context.Background(), "method", "get_all_models"), request4)
	assert.Nil(suite.T(), err)
	modelIds := []string{}
	for _, model := range response4.ListOfModels {
		modelIds = append(modelIds, model.ModelId)
	}

	assert.True(suite.T(), slices.Equal(expectedModelIds, modelIds))
}

func (suite *DaemonServiceSuite) TestDaemonService_ManageUpdateStatusWorkers() {
	duration := time.Second * 12
	deadline := time.Now().Add(duration)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		suite.T().Logf("context done %v", ctx.Err())
	case <-time.After(duration):
		suite.T().Logf("operation timed out after: %v", duration)
	}

	for _, modelKey := range suite.pendingModelKeys {
		modelData, ok, err := suite.modelStorage.Get(modelKey)
		assert.Nil(suite.T(), err)
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), Status_VALIDATED, modelData.Status)
	}
}

func (suite *DaemonServiceSuite) TestDaemonService_UnimplementedDaemonService() {
	response1, err := suite.unimplementedDaemonService.CreateModel(context.TODO(), &NewModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response1.Status)

	_, err = suite.unimplementedDaemonService.ValidateModelPrice(context.TODO(), &AuthValidateRequest{})
	assert.NotNil(suite.T(), err)

	response2, err := suite.unimplementedDaemonService.ValidateModel(context.TODO(), &AuthValidateRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response2.Status)

	_, err = suite.unimplementedDaemonService.TrainModelPrice(context.TODO(), &CommonRequest{})
	assert.NotNil(suite.T(), err)

	response3, err := suite.unimplementedDaemonService.TrainModel(context.TODO(), &CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response3.Status)

	response4, err := suite.unimplementedDaemonService.DeleteModel(context.TODO(), &CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response4.Status)

	response5, err := suite.unimplementedDaemonService.GetAllModels(context.TODO(), &AllModelsRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), []*ModelResponse{}, response5.ListOfModels)

	response6, err := suite.unimplementedDaemonService.GetModel(context.TODO(), &CommonRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response6.Status)

	response7, err := suite.unimplementedDaemonService.UpdateModel(context.Background(), &UpdateModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), Status_ERRORED, response7.Status)

	_, err = suite.unimplementedDaemonService.GetTrainingMetadata(context.Background(), &emptypb.Empty{})
	assert.NotNil(suite.T(), err)

	_, err = suite.unimplementedDaemonService.GetMethodMetadata(context.TODO(), &MethodMetadataRequest{})
	assert.NotNil(suite.T(), err)
}
