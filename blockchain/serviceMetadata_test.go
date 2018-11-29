package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/magiconair/properties/assert"
	"github.com/singnet/snet-daemon/config"
	"math/big"
	"strings"
	"testing"
)

var daemonEndPoint string = config.GetString(config.DaemonEndPoint)
var testJsonData = "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grpc\", \"service_type\": \"grpc\", \"payment_expiration_threshold\": 40320, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\", \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\", \"pricing\": {\"price_model\": \"fixed_price\", \"price_in_cogs\": 12000000}, \"groups\": [{\"group_name\": \"default_group\", \"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"0xD6C6344f1D122dC6f4C1782A4622B683b9008081\"}], \"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"" + daemonEndPoint + "\"}]}"

func TestAllGetterMethods(t *testing.T) {
	println(testJsonData)
	metaData, err := InitServiceMetaDataFromJson(testJsonData)
	assert.Equal(t, err, nil)
	assert.Equal(t, metaData.GetDaemonGroupName(), "default_group")
	assert.Equal(t, metaData.GetVersion(), 1)
	assert.Equal(t, metaData.GetDisplayName(), "Example1")
	assert.Equal(t, metaData.GetServiceType(), "grpc")
	assert.Equal(t, metaData.GetWireEncoding(), "grpc")
	assert.Equal(t, metaData.GetDaemonEndPoint(), ""+daemonEndPoint)
	assert.Equal(t, metaData.GetPaymentAddress(), common.HexToAddress("0xD6C6344f1D122dC6f4C1782A4622B683b9008081"))
	assert.Equal(t, metaData.GetPaymentExpirationThreshold(), big.NewInt(40320))
	assert.Equal(t, metaData.GetPriceInCogs(), big.NewInt(12000000))
	assert.Equal(t, metaData.GetMpeAddress(), common.HexToAddress("0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E"))
	encodedStr, _ := ConvertBase64Encoding("nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=")
	assert.Equal(t, metaData.GetDaemonGroupID(), encodedStr)

}

func TestServiceMetadata_GetDaemonGroupName(t *testing.T) {
	//Change the Daemon end point in json to not match the daemon end point in config
	_, err := InitServiceMetaDataFromJson(strings.Replace(testJsonData, ""+daemonEndPoint, "127.0.0.2:8080", -1))
	assert.Equal(t, err.Error(), "unable to determine Daemon Group Name, DaemonEndPoint "+daemonEndPoint)

}

func TestServiceMetadata_GetDaemonGroupID(t *testing.T) {
	//Change the GroupName in Groups
	_, err := InitServiceMetaDataFromJson(strings.Replace(testJsonData, "default_group", "default_group1", 1))
	assert.Equal(t, err.Error(), "unable to determine the Daemon Group ID or the Recipient Payment Address, Daemon Group Name default_group")

}

func TestInitServiceMetaDataFromJson(t *testing.T) {
	//Parse Bad JSON
	_, err := InitServiceMetaDataFromJson(strings.Replace(testJsonData, "{", "", 1))
	assert.Equal(t, err.Error(), "invalid character ':' after top-level value")

}

func TestReadServiceMetaDataFromLocalFile(t *testing.T) {
	metadata, err := readServiceMetaDataFromLocalFile("../service_metadata.json")
	assert.Equal(t, err, nil)
	assert.Equal(t, metadata.Version, 1)

}
