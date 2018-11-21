package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

type ServiceMetadata struct {
	Version                   int    `json:"version"`
	DisplayName               string `json:"display_name"`
	Encoding                  string `json:"encoding"`
	ServiceType               string `json:"service_type"`
	PaymentEpirationThreshold int64  `json:"payment_expiration_threshold"`
	ModelIpfsHash             string `json:"model_ipfs_hash"`
	MpeAddress                string `json:"mpe_address"`
	Pricing                   struct {
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

var metaData *ServiceMetadata

func GetDaemonGroupID() [32]byte {
	var byte32 [32]byte
	groupName := GetDaemonGroupName()
	for _, group := range metaData.Groups {
		if strings.Compare(groupName, group.GroupName) == 0 {
			groupID := group.GroupID
			metaData.DeamonReplicaGroupID = groupID
			metaData.DeamonGroupName = group.GroupName
			return getbase64Encoding(groupID)
		}
	}
	log.WithField("GetDaemonGroupID",
		"Group ID could not be retrieved, check if the daemon end point in config " +
			"matches the end point from metadata").Panic("serviceMetadata.GetDaemonGroupID")

	return byte32
}

func getbase64Encoding(str string) [32]byte {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		log.WithError(err).Panic("Error trying to base64.StdEncoding.DecodeString")
	}
	var byte32 [32]byte
	copy(byte32[:], data[:])
	return byte32

}

func GetPaymentAddress() string {
	var paymentAddress string
	groupName := GetDaemonGroupName()
	for _, group := range metaData.Groups {
		if strings.Compare(groupName, group.GroupName) == 0 {
			paymentAddress = group.PaymentAddress
			return paymentAddress
		}
	}
	log.WithField("GetPaymentAddress",
		"Payment Address could not be retrieved, check if the daemon end point in config matches " +
			"the end point from metadata").Panic("serviceMetadata.GetPaymentAddress")
	return paymentAddress
}

func SetServiceMetaData(hash string) {
	jsondata := ipfsutils.GetIpfsFile(hash)
	metaData = new(ServiceMetadata)
	json.Unmarshal([]byte(jsondata), &metaData)
}

func SetServiceMetaDataThroughJSON(jsondata string) {
	metaData = new(ServiceMetadata)
	json.Unmarshal([]byte(jsondata), &metaData)
}

func GetmpeAddress() string {
	return metaData.MpeAddress
}

func GetPaymentExpirationThreshold() int64 {
	return metaData.PaymentEpirationThreshold
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

			return groupName
		}
	}
	log.WithField("GetDaemonGroupName",
		"Daemon group name could not be determined"+
			", check if the daemon end point in config matches the end point from metadata").Panic("serviceMetadata.GetPaymentAddress")

	return groupName
}

func GetWireEncoding() string {

	return metaData.Encoding

}

func GetServiceType() string {
	return metaData.ServiceType
}

func GetDisplayName() string {

	return metaData.DisplayName

}