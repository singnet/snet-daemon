package tests

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/training"
)

type DaemonServiceSuite struct {
	suite.Suite
	modelStorage        *training.ModelStorage
	userModelStorage    *training.ModelUserStorage
	pendingModelStorage *training.PendingModelStorage
	daemonService       training.DaemonServer
	modelKeys           []*training.ModelKey
	grpcServer          *grpc.Server
}

var (
	testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"test_org_id\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",  \"license_server_endpoints\": [\"https://licensendpoint:8082\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonServiceData  = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "
)

func (suite *DaemonServiceSuite) SetupSuite() {
	suite.setupTestConfig()

	modelKeys, modelStorage, userModelStorage, pendingModelStorage, publicModelStorage := suite.createTestModels()
	suite.modelKeys = modelKeys
	suite.modelStorage = modelStorage
	suite.userModelStorage = userModelStorage
	suite.pendingModelStorage = pendingModelStorage

	serviceMetadata, err := blockchain.InitServiceMetaDataFromJson([]byte(testJsonServiceData))
	orgMetadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	if err != nil {
		zap.L().Fatal("Error in initinalize organization metadata from json", zap.Error(err))
	}

	suite.daemonService = training.NewTrainingService(
		nil,
		serviceMetadata,
		orgMetadata,
		modelStorage,
		userModelStorage,
		pendingModelStorage,
		publicModelStorage,
	)

	address := "localhost:5001"
	suite.grpcServer = startTestService(address)
}

func TestDaemonServiceSuite(t *testing.T) {
	suite.Run(t, new(DaemonServiceSuite))
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

func (suite *DaemonServiceSuite) createTestModels() ([]*training.ModelKey,
	*training.ModelStorage, *training.ModelUserStorage, *training.PendingModelStorage, *training.PublicModelStorage) {
	memStorage := storage.NewMemStorage()
	modelStorage := training.NewModelStorage(memStorage)
	userModelStorage := training.NewUserModelStorage(memStorage)
	pendingModelStorage := training.NewPendingModelStorage(memStorage)
	publicModelStorage := training.NewPublicModelStorage(memStorage)

	modelA := &training.ModelData{
		IsPublic:            true,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              training.Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "1",
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
		ModelId:        "1",
	}

	modelB := &training.ModelData{
		IsPublic:            true,
		ModelName:           "testModel",
		AuthorizedAddresses: []string{},
		Status:              training.Status_CREATED,
		CreatedByAddress:    "address",
		ModelId:             "1",
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
		ModelId:        "2",
	}

	modelStorage.Put(modelAKey, modelA)
	modelStorage.Put(modelBKey, modelB)
	modelKeys := []*training.ModelKey{modelAKey, modelBKey}

	key := &training.PendingModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	pendingModelsData := &training.PendingModelData{
		ModelIDs: []string{"1", "2"},
	}

	pendingModelStorage.Put(key, pendingModelsData)

	return modelKeys, modelStorage, userModelStorage, pendingModelStorage, publicModelStorage
}

func (suite *DaemonServiceSuite) TestRunDaemonService() {

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

	for _, modelKey := range suite.modelKeys {
		modelData, ok, err := suite.modelStorage.Get(modelKey)
		assert.Nil(suite.T(), err)
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), training.Status_VALIDATED, modelData.Status)
	}
}
