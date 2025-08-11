package metrics

import (
	"math/big"
	"strconv"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	timeFormat = "2006-01-02 15:04:05.999999999"
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
	UserName            string
	PaymentMode         string
	UserAddress         string
}

type ChannelStats struct {
	OrganizationID   string
	ServiceID        string
	GroupID          string
	AuthorizedAmount *big.Int
	FullAmount       *big.Int
	ChannelId        *big.Int
	Nonce            *big.Int
}

func BuildCommonStats(receivedTime time.Time, methodName string) *CommonStats {
	commonStats := &CommonStats{
		ID:                  GenXid(),
		GroupID:             daemonGroupId,
		RequestReceivedTime: receivedTime.UTC().Format(timeFormat),
		OrganizationID:      config.GetString(config.OrganizationId),
		ServiceID:           config.GetString(config.ServiceId),
		ServiceMethod:       methodName,
		Version:             config.GetVersionTag(),
	}
	return commonStats

}

// Response stats that will be captured and published
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
	UserName                   string `json:"username"`
	Operation                  string `json:"operation"`
	UsageType                  string `json:"usage_type"`
	Status                     string `json:"status"`
	StartTime                  string `json:"start_time"`
	EndTime                    string `json:"end_time"`
	UsageValue                 int    `json:"usage_value"`
	TimeZone                   string `json:"time_zone"`
	PaymentMode                string `json:"payment_mode"`
	UserAddress                string `json:"user_address"`
}

// Publish response received as a payload for reporting /metrics analysis
// If there is an error in the response received from the service, then send out a notification as well.
func PublishResponseStats(commonStats *CommonStats, duration time.Duration, err error, block *big.Int) bool {
	response := createResponseStats(commonStats, duration, err)
	return Publish(response, config.GetString(config.MeteringEndpoint)+"/metering/usage", commonStats, block)
}

func createResponseStats(commonStat *CommonStats, duration time.Duration, err error) *ResponseStats {
	currentTime := time.Now().UTC().Format(timeFormat)

	response := &ResponseStats{
		Type:                       "response",
		RegistryAddressKey:         config.GetRegistryAddress(),
		EthereumJsonRpcEndpointKey: config.GetBlockChainHTTPEndPoint(),
		RequestID:                  commonStat.ID,
		ResponseTime:               strconv.FormatFloat(duration.Seconds(), 'f', 4, 64),
		GroupID:                    daemonGroupId,
		OrganizationID:             commonStat.OrganizationID,
		ServiceID:                  commonStat.ServiceID,
		ServiceMethod:              commonStat.ServiceMethod,
		RequestReceivedTime:        commonStat.RequestReceivedTime,
		ResponseSentTime:           currentTime,
		ErrorMessage:               getErrorMessage(err),
		ResponseCode:               getErrorCode(err),
		Version:                    commonStat.Version,
		ClientType:                 commonStat.ClientType,
		UserDetails:                commonStat.UserDetails,
		UserAgent:                  commonStat.UserAgent,
		ChannelId:                  commonStat.ChannelId,
		UserName:                   commonStat.UserName,
		StartTime:                  commonStat.RequestReceivedTime,
		EndTime:                    currentTime,
		Status:                     getStatus(err),
		UsageValue:                 1,
		UsageType:                  "apicall",
		Operation:                  "read",
		PaymentMode:                commonStat.PaymentMode,
		UserAddress:                commonStat.UserAddress,
	}
	return response
}

func getStatus(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
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
