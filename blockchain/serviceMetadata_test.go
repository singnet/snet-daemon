package blockchain

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/magiconair/properties/assert"
	"math/big"
	"testing"
)

func TestAllGetterMethods(t *testing.T) {
	jsondata := "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grcp\", \"service_type\": \"grcp\", \"payment_expiration_threshold\": 40320, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\", \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\", \"pricing\": {\"price_model\": \"fixed_price\", \"price_in_cogs\": 12000000}, \"groups\": [{\"group_name\": \"default_group\", \"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"0xD6C6344f1D122dC6f4C1782A4622B683b9008081\"}], \"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"127.0.0.1:8080\"}]}"
	metaData, err := InitServiceMetaDataFromJson(jsondata)
	assert.Equal(t, err, nil)
	assert.Equal(t, metaData.GetDaemonGroupName(), "default_group")
	assert.Equal(t, metaData.GetDaemonEndPoint(), "127.0.0.1:8080")
	assert.Equal(t, metaData.GetPaymentAddress(), common.HexToAddress("0xD6C6344f1D122dC6f4C1782A4622B683b9008081"))
	assert.Equal(t, metaData.PaymentExpirationThreshold, int64(40320))
	assert.Equal(t, metaData.GetPriceInCogs(), big.NewInt(12000000))
	assert.Equal(t, metaData.GetMpeAddress(), common.HexToAddress("0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E"))
	assert.Equal(t, metaData.GetDaemonGroupID(), convertBase64Encoding("nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U="))

}

func TestDaemonMetaDataErrors(t *testing.T) {
	jsondata := "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grcp\", \"service_type\": \"grcp\", \"payment_expiration_threshold\": 40320, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\", \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\", \"pricing\": {\"price_model\": \"fixed_price\", \"price_in_cogs\": 12000000}, \"groups\": [{\"group_name\": \"default_group\", \"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"0xD6C6344f1D122dC6f4C1782A4622B683b9008081\"}], \"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"127.0.0.2:8080\"}]}"
	metaData := new(ServiceMetadata)
	json.Unmarshal([]byte(jsondata), &metaData)

	err := setDaemonEndPoint(metaData)
	assert.Equal(t, err, nil)

	//TO DO , check all error messages + Panic test cases

}
