package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	"math/big"
	"strings"
)


type ServiceMetadata struct {
	Version                    int    `json:"version"`
	DisplayName                string `json:"display_name"`
	Encoding                   string `json:"encoding"`
	ServiceType                string `json:"service_type"`
	PaymentEpirationThreshold int64    `json:"payment_expiration_threshold"`
	ModelIpfsHash              string `json:"model_ipfs_hash"`
	MpeAddress                 string `json:"mpe_address"`
	Pricing                    struct {
		PriceModel string `json:"price_model"`
		PriceInCogs      big.Int    `json:"price_in_cogs"`
	} `json:"pricing"`
	Groups [] struct {
		GroupName      string `json:"group_name"`
		GroupID        string `json:"group_id"`
		PaymentAddress string `json:"payment_address"`
	} `json:"groups"`
	Endpoints [] struct {
		GroupName string `json:"group_name"`
		Endpoint  string `json:"endpoint"`
	} `json:"endpoints"`
	DeamonReplicaGroupID string
	DeamonGroupName      string
	DaemonEndPoint       string
}
var metaData *ServiceMetadata

func GetDaemonGroupID()  [32]byte {
	groupID:= "0"
	groupName:= GetDaemonGroupName()
	for _,group:= range metaData.Groups {
		if strings.Compare(groupName,group.GroupName) == 0 {
			groupID = group.GroupID
			metaData.DeamonReplicaGroupID = groupID
			metaData.DeamonGroupName = group.GroupName
			break
		}
	}

	data,err := base64.StdEncoding.DecodeString(groupID)
	if(err != nil) {

	}
	var byte32 [32]byte
	copy(byte32[:], data[:])
	return byte32
}

func GetPaymentAddress()  string {
	paymentAddress := "0" //to continue with current testing
	groupName:= GetDaemonGroupName()
	for _,group:= range metaData.Groups {
		if strings.Compare(groupName,group.GroupName) == 0 {
			paymentAddress = group.PaymentAddress
		}
	}
	return paymentAddress
}


func SetServiceMetaData(hash string) {
	jsondata := ipfsutils.GetIpfsFile(hash)
	metaData =  new(ServiceMetadata)
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

func GetPriceinCogs() big.Int  {
	return metaData.Pricing.PriceInCogs
}
//Get the group name based on end point
func GetDaemonGroupName() string {
	groupName := "0"
	for _,endpoints := range metaData.Endpoints {
		if strings.Compare(config.GetString(config.DaemonEndPoint),endpoints.Endpoint) ==0 {
			groupName= endpoints.GroupName
		}
	}
	return groupName
}

func GetWireEncoding() string {
	/*if metaData != nil {
		return metaData.Encoding
	}*/
	return metaData.Encoding
	//return config.GetString(config.WireEncodingKey)
}

func GetVersion() string {

	return metaData.Encoding

}

func GetServiceType() string {
	/*if metaData != nil {
		return metaData.ServiceType
	}*/
	return metaData.ServiceType
	//return config.GetString(config.ServiceTypeKey)
}

func GetDisplayName() string {

	return metaData.DisplayName

}
