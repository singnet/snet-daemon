package metrics

import (
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
}

func BuildCommonStats(receivedTime time.Time, methodName string) *CommonStats {
	commonStats := &CommonStats{
		ID:                  GenXid(),
		GroupID:             daemonGroupId,
		RequestReceivedTime: receivedTime.String(),
		DaemonEndPoint:      config.GetString(config.DaemonEndPoint),
		OrganizationID:      config.GetString(config.OrganizationId),
		ServiceID:           config.GetString(config.ServiceId),
		ServiceMethod:       methodName,
	}
	return commonStats

}

//Response stats that will be captured and published
type ResponseStats struct {
	RequestID           string `json:"request_id"`
	OrganizationID      string `json:"organization_id"`
	ServiceID           string `json:"service_id"`
	GroupID             string `json:"Group_id"`
	DaemonEndPoint      string `json:"Daemon_end_point"`
	ServiceMethod       string `json:"service_method"`
	ResponseSentTime    string `json:"response_sent_time"`
	RequestReceivedTime string `json:"request_arrival_time"`
	ResponseTime        string `json:"response_time"`
	ResponseCode        string `json:"response_code"`
	ErrorMessage        string `json:"error_message"`
}

func PublishResponseStats(commonStats *CommonStats, duration time.Duration, err error) bool {
	response := createResponseStats(commonStats, duration, err)
	return Publish(response, config.GetString(config.MonitoringServiceEndpoint)+"/event")
}

func createResponseStats(commonStat *CommonStats, duration time.Duration, err error) *ResponseStats {
	response := &ResponseStats{
		RequestID:           commonStat.ID,
		ResponseTime:        duration.String(),
		GroupID:             daemonGroupId,
		DaemonEndPoint:      commonStat.DaemonEndPoint,
		OrganizationID:      commonStat.OrganizationID,
		ServiceID:           commonStat.ServiceID,
		ServiceMethod:       commonStat.ServiceMethod,
		RequestReceivedTime: commonStat.RequestReceivedTime,
		ResponseSentTime:    time.Now().String(),
		ErrorMessage:        getErrorMessage(err),
		ResponseCode:        getErrorCode(err),
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
