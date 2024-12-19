package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/training"
)

func setupTestConfig() {
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

var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"test_org_id\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",  \"license_server_endpoints\": [\"https://licensendpoint:8082\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"

func addTestModels() (*training.ModelKey, *training.ModelKey, map[string]training.ModelData, *training.ModelStorage) {
	memStorage := storage.NewMemStorage()
	modelStorage := training.NewModelStorage(memStorage)

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

	models := map[string]training.ModelData{
		"1": *modelA,
		"2": *modelB,
	}

	modelIdsData := "{DATA:[1, 2]}"

	key := &training.TrainingValidatingModelKey{
		OrganizationId: "test_org_id",
		ServiceId:      "service_id",
		GroupId:        "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
	}

	memStorage.Put(key.String(), modelIdsData)

	return modelAKey, modelBKey, models, modelStorage
}

func TestRunDaemonService(t *testing.T) {
	setupTestConfig()
	address := "localhost:5001"

	modelAKey, modelBKey, _, modelStorage := addTestModels()
	startTestService(address)

	orgMetadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	assert.Nil(t, err)

	ds := training.NewDaemonsService(
		nil, orgMetadata, nil, modelStorage, nil, "http://localhost:5001", nil, nil,
	)
	assert.NotNil(t, ds)

	duration := time.Second * 10
	deadline := time.Now().Add(duration)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	ds.ManageUpdateModelStatusWorkers(ctx, time.Second*3, "test_org_id", "service_id", "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=")

	select {
	case <-ctx.Done():
		t.Logf("Context done: %v", ctx.Err())
	case <-time.After(duration):
		t.Logf("Operation timed out after %v", duration)
	}

	model1Data, ok, err := modelStorage.Get(modelAKey)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, training.Status_VALIDATED, model1Data.Status)

	model2Data, ok, err := modelStorage.Get(modelBKey)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, training.Status_VALIDATED, model2Data.Status)
}
