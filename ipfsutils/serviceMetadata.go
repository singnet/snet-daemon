package ipfsutils

import (
	"encoding/json"
	"github.com/singnet/snet-daemon/config"
	"math/big"
	"strings"
)

type service_metadata struct {
	Version                    int    `json:"version"`
	DisplayName                string `json:"display_name"`
	Encoding                   string `json:"encoding"`
	ServiceType                string `json:"service_type"`
	PaymentExpirationThreshold int    `json:"payment_expiration_threshold"`
	ModelIpfsHash              string `json:"model_ipfs_hash"`
	MpeAddress                 string `json:"mpe_address"`
	Pricing                    struct {
		PriceModel  string   `json:"price_model"`
		PriceInCogs *big.Int `json:"price_in_cogs"`
	} `json:"pricing"`
	Groups []struct {
		GroupName      string `json:"group_name"`
		GroupID        string `json:"group_id"`
		PaymentAddress string `json:"payment_address"`
	} `json:"groups"`
	Endpoints []struct {
		GroupName string `json:"group_name"`
		Endpoint  string `json:"endpoint"`
	} `json:"endpoints"`
	DeamonReplicaGroupID string
	DeamonGroupName      string
	DaemonEndPoint       string
}

var metaData *service_metadata

func GetDaemonGroupID() [32]byte {
	groupID := "0"
	groupName := GetDaemonGroupName()
	for _, group := range metaData.Groups {
		if strings.Compare(groupName, group.GroupName) == 0 {
			groupID = group.GroupID
			metaData.DeamonReplicaGroupID = groupID
			metaData.DeamonGroupName = group.GroupName
		}
	}
	return StringToBytes32(groupID)
}

func GetPaymentAddress() string {
	paymentAddress := "0" //to continue with current testing
	groupName := GetDaemonGroupName()
	for _, group := range metaData.Groups {
		if strings.Compare(groupName, group.GroupName) == 0 {
			paymentAddress = group.PaymentAddress
		}
	}
	return paymentAddress
}

func StringToBytes32(str string) [32]byte {

	var byte32 [32]byte
	copy(byte32[:], []byte(str))

	return byte32
}

func SetServiceMetaData(hash string) {
	jsondata := GetIpfsFile(hash)
	metaData = new(service_metadata)
	json.Unmarshal([]byte(jsondata), &metaData)
}

func SetServiceMetaDataThroughJSON(jsondata string) {
	metaData = new(service_metadata)
	json.Unmarshal([]byte(jsondata), &metaData)
}

func GetPaymentExpirationThreshold() int {
	return metaData.PaymentExpirationThreshold
}

func GetPriceinCogs() *big.Int {
	return metaData.Pricing.PriceInCogs
}

//Get the group name based on end point
func GetDaemonGroupName() string {
	groupName := "0"
	for _, endpoints := range metaData.Endpoints {
		if strings.Compare(config.GetString(config.DaemonEndPoint), endpoints.Endpoint) == 0 {
			groupName = endpoints.GroupName
		}
	}
	return groupName
}
