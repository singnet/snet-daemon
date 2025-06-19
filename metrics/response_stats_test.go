package metrics

import (
	"fmt"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCreateResponseStats(t *testing.T) {
	arrivalTime := time.Now()
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	commonStat.ClientType = "snet-cli"
	commonStat.UserDetails = "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	commonStat.UserAgent = "python/cli"
	commonStat.ChannelId = "1"
	commonStat.PaymentMode = "freecall"
	commonStat.UserAddress = "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	response := createResponseStats(commonStat, time.Duration(1234566000), nil)
	assert.Equal(t, response.RequestID, commonStat.ID)
	assert.Equal(t, response.Version, commonStat.Version)
	assert.Equal(t, response.GroupID, daemonGroupId)
	assert.Equal(t, response.ResponseTime, "1.2346")
	assert.Equal(t, response.Type, "response")
	assert.NotEqual(t, response.ResponseSentTime, "")
	assert.Equal(t, response.ClientType, "snet-cli")
	assert.Equal(t, response.UserDetails, "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
	assert.Equal(t, response.UserAgent, "python/cli")
	assert.Equal(t, response.ChannelId, "1")
	assert.NotNil(t, response.EndTime)
	assert.Equal(t, response.PaymentMode, "freecall")
	assert.Equal(t, response.UserAddress, "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
}

func TestGetErrorMessage(t *testing.T) {
	err := fmt.Errorf("test Error")
	msg := getErrorMessage(err)
	assert.Equal(t, msg, "test Error")
	assert.Equal(t, getErrorMessage(nil), "")

}

func TestGetErrorCode(t *testing.T) {
	err := fmt.Errorf("test Error")
	code := getErrorCode(err)
	assert.Equal(t, code, "Unknown")
	assert.Equal(t, getErrorCode(nil), "OK")
}

func TestJsonCreated(t *testing.T) {

	tim := time.Now()
	zone, _ := tim.Zone()
	payload := &ResponseStats{

		Type:      "grpc",
		ChannelId: "123",

		RegistryAddressKey:         "",
		EthereumJsonRpcEndpointKey: "",
		RequestID:                  "",
		OrganizationID:             config.GetString(config.OrganizationId),
		GroupID:                    "",
		ServiceMethod:              "",
		ResponseSentTime:           "",
		RequestReceivedTime:        "",
		ResponseTime:               "",
		ResponseCode:               "",
		ErrorMessage:               "",
		Version:                    "",
		ClientType:                 "",
		UserDetails:                "",
		UserAgent:                  "",
		UserName:                   "whateverDappPasses",
		Operation:                  "",
		UsageType:                  "",
		Status:                     "",
		StartTime:                  "",
		EndTime:                    "",
		UsageValue:                 1,
		TimeZone:                   zone,
	}
	jsonBytes, err := ConvertStructToJSON(payload)
	assert.NotNil(t, jsonBytes)
	assert.Contains(t, string(jsonBytes), "whateverDappPasses")
	assert.Nil(t, err)

}
