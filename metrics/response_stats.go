package metrics

import (
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

type CommonStats struct {
	ID                  string
	ServiceMethod       string
	RequestReceivedTime string
	OrganizationID      string
	ServiceID           string
	GroupID             string
	DaemonEndPoint      string
	Version             string
	ClientType          string
	UserDetails         string
	UserAgent           string
	ChannelId           string
}

func BuildCommonStats(receivedTime time.Time, methodName string) *CommonStats {
	commonStats := &CommonStats{
		ID:                  GenXid(),
		GroupID:             daemonGroupId,
		RequestReceivedTime: receivedTime.String(),
		OrganizationID:      config.GetString(config.OrganizationId),
		ServiceID:           config.GetString(config.ServiceId),
		ServiceMethod:       methodName,
		Version:             config.GetVersionTag(),
		ClientType:          "",
		UserDetails:         "",
		UserAgent:           "",
	}
	return commonStats

}

//Response stats that will be captured and published
type ResponseStats struct {
	Type                       string `json:"type"`
	RegistryAddressKey         string `json:"registry_address_key"`
	EthereumJsonRpcEndpointKey string `json:"ethereum_json_rpc_endpoint"`
	RequestID                  string `json:"request_id"`
	OrganizationID             string `json:"organization_id"`
	ServiceID                  string `json:"service_id"`
	GroupID                    string `json:"group_id"`
	ServiceMethod              string `json:"service_method"`
	ResponseSentTime           string `json:"response_sent_time"`
	RequestReceivedTime        string `json:"request_received_time"`
	ResponseTime               string `json:"response_time"`
	ResponseCode               string `json:"response_code"`
	ErrorMessage               string `json:"error_message"`
	Version                    string `json:"version"`
	ClientType                 string `json:"client_type"`
	UserDetails                string `json:"user_details"`
	UserAgent                  string `json:"user_agent"`
	ChannelId                  string `json:"channel_id"`
}

//Publish response received as a payload for reporting /metrics analysis
//If there is an error in the response received from the service, then send out a notification as well.
func PublishResponseStats(commonStats *CommonStats, duration time.Duration, err error) bool {
	response := createResponseStats(commonStats, duration, err)
	return Publish(response, config.GetString(config.MonitoringServiceEndpoint)+"/event")
}

func createResponseStats(commonStat *CommonStats, duration time.Duration, err error) *ResponseStats {
	response := &ResponseStats{
		Type:                       "response",
		RegistryAddressKey:         config.GetRegistryAddress(),
		EthereumJsonRpcEndpointKey: config.GetBlockChainEndPoint(),
		RequestID:                  commonStat.ID,
		ResponseTime:               strconv.FormatFloat(duration.Seconds(), 'f', 4, 64),
		GroupID:                    daemonGroupId,
		OrganizationID:             commonStat.OrganizationID,
		ServiceID:                  commonStat.ServiceID,
		ServiceMethod:              commonStat.ServiceMethod,
		RequestReceivedTime:        commonStat.RequestReceivedTime,
		ResponseSentTime:           time.Now().String(),
		ErrorMessage:               getErrorMessage(err),
		ResponseCode:               getErrorCode(err),
		Version:                    commonStat.Version,
		ClientType:                 commonStat.ClientType,
		UserDetails:                commonStat.UserDetails,
		UserAgent:                  commonStat.UserAgent,
		ChannelId:                  commonStat.ChannelId,
	}
	return response
}

func getErrorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func getErrorCode(err error) string {
	statCode, ok := status.FromError(err)
	if !ok {
		return codes.Unknown.String()
	}
	return statCode.Code().String()

}
