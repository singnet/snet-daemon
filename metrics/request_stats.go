package metrics

import (
	"github.com/singnet/snet-daemon/v6/config"
)

// Request stats that will be captured
type RequestStats struct {
	Type                       string `json:"type"`
	RegistryAddressKey         string `json:"registry_address_key"`
	EthereumJsonRpcEndpointKey string `json:"ethereum_json_rpc_endpoint"`
	RequestID                  string `json:"request_id"`
	InputDataSize              string `json:"input_data_size"`
	ServiceMethod              string `json:"service_method"`
	RequestReceivedTime        string `json:"request_received_time"`
	OrganizationID             string `json:"organization_id"`
	ServiceID                  string `json:"service_id"`
	GroupID                    string `json:"group_id"`
	DaemonEndPoint             string `json:"daemon_end_point"`
	Version                    string `json:"version"`
	ClientType                 string `json:"client_type"`
	UserDetails                string `json:"user_details"`
	UserAgent                  string `json:"user_agent"`
	ChannelId                  string `json:"channel_id"`
}

//func (request *RequestStats) setDataFromContext(md metadata.MD) {
//	request.InputDataSize = strconv.FormatUint(GetSize(md), 10)
//}

func createRequestStat(commonStat *CommonStats) *RequestStats {
	request := &RequestStats{
		Type:                       "request",
		RegistryAddressKey:         config.GetRegistryAddress(),
		EthereumJsonRpcEndpointKey: config.GetBlockChainHTTPEndPoint(),
		RequestID:                  commonStat.ID,
		GroupID:                    commonStat.GroupID,
		DaemonEndPoint:             commonStat.DaemonEndPoint,
		OrganizationID:             commonStat.OrganizationID,
		ServiceID:                  commonStat.ServiceID,
		RequestReceivedTime:        commonStat.RequestReceivedTime,
		ServiceMethod:              commonStat.ServiceMethod,
		Version:                    commonStat.Version,
		ClientType:                 commonStat.ClientType,
		UserDetails:                commonStat.UserDetails,
		UserAgent:                  commonStat.UserAgent,
		ChannelId:                  commonStat.ChannelId,
	}
	return request
}
