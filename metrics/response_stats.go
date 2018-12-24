package metrics

import (
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

//Response stats that will be captured and published
type ResponseStats struct {
	RequestID        string `json:"request_id"`
	OrganizationID   string `json:"organization_id"`
	ServiceID        string `json:"service_id"`
	GroupID          string `json:"Group_id"`
	DaemonEndPoint   string `json:"Daemon_end_point"`
	ResponseSentTime string `json:"response_sent_time"`
	ResponseTime     string `json:"response_time"`
	ResponseCode     string `json:"response_code"`
	ErrorMessage     string `json:"error_message"`
}

func PublishResponseStats(reqId string, grpId string, duration time.Duration, err error) bool {
	response := createResponseStats(reqId, grpId, duration, err)
	return Publish(response, config.GetString(config.MonitoringServiceEndpoint)+"/event")
}

func createResponseStats(reqId string, grpId string, duration time.Duration, err error) *ResponseStats {
	response := &ResponseStats{
		RequestID:      reqId,
		ResponseTime:   duration.String(),
		GroupID:        grpId,
		DaemonEndPoint: config.GetString(config.DaemonEndPoint),
		OrganizationID: config.GetString(config.OrganizationId),
		ServiceID:      config.GetString(config.ServiceId),
		ErrorMessage:   getErrorMessage(err),
		ResponseCode:   getErrorCode(err),
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
