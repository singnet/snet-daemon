package blockchain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testJsonOrgGroupData = "{\"org_name\":\"organization_name\",\"org_id\":\"org_id1\",\"groups\": [   {\"group_name\":\"group1\",\"group_id\":\"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"payment_address\":\"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",\"payment_channel_storage_type\":\"etcd\",\"payment_channel_storage_client\": {\"connection_timeout\":\"5s\",\"request_timeout\":\"3s\",\"endpoints\": [\"http://127.0.0.1:2379\"    ]    }   },   {\"group_name\":\"default_group\",\"group_id\":\"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"payment_address\":\"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",\"payment_channel_storage_type\":\"etcd\",\"payment_channel_storage_client\": {\"connection_timeout\":\"5s\",\"request_timeout\":\"3s\",\"endpoints\": [\"http://127.0.0.1:2479\"    ]   }   } ] }"

func TestOrganizationMetaData_GetGroupId(t *testing.T) {

}

func TestInitOrganizationMetaDataFromJson(t *testing.T) {

}

func TestOrganizationMetaData_GetPaymentAddress(t *testing.T) {

}

func TestGetOrganizationMetaData(t *testing.T) {
	metadata, err := InitOrganizationMetaDataFromJson(testJsonOrgGroupData)
	assert.Nil(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "organization_name", metadata.OrgName)
	address := metadata.GetPaymentAddress()
	assert.Equal(t, "0x671276c61943A35D5F230d076bDFd91B0c47bF09", address)

}

func TestGetOrganizationMetaDataFromIPFS(t *testing.T) {

}

func Test_getOrgMetadataURI(t *testing.T) {

}
