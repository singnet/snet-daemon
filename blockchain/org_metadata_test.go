package blockchain

import (
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
)

var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",  \"license_server_endpoints\": [\"https://licensendpoint:8082\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"

func TestGetOrganizationMetaData(t *testing.T) {
	metadata, err := InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	assert.Nil(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "organization_name", metadata.OrgName)
	address := metadata.GetPaymentAddress()
	assert.Equal(t, "0x671276c61943A35D5F230d076bDFd91B0c47bF09", address.Hex())
	assert.Equal(t, "http://127.0.0.1:2379", metadata.GetPaymentStorageEndPoints()[0])
	assert.Equal(t, "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=", metadata.GetGroupIdString())
	assert.Equal(t, big.NewInt(40320), metadata.GetPaymentExpirationThreshold())
	grpId, _ := utils.ConvertBase64Encoding("99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=")
	assert.Equal(t, grpId, metadata.GetGroupId())
	assert.Equal(t, 15*time.Second, metadata.GetConnectionTimeOut())
	assert.Equal(t, 13*time.Second, metadata.GetRequestTimeOut())
	assert.Equal(t, "https://licensendpoint:8082", metadata.GetLicenseEndPoints()[0])
}

func TestGetOrganizationMetaDataForError(t *testing.T) {
	metadata, err := InitOrganizationMetaDataFromJson([]byte("bad json"))
	assert.Nil(t, metadata)
	assert.NotNil(t, err)

	config.Vip().Set(config.DaemonGroupName, "unknow")
	if metadata, err = InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData)); err != nil {
		assert.Nil(t, metadata)
		assert.Equal(t, "group name unknow in config is invalid, there was no group found with this name in the metadata", err.Error())
	}
	config.Vip().Set(config.DaemonGroupName, "default_group")
}
