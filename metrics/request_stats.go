package metrics

import (
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strconv"
)

//Request stats that will be captured
type RequestStats struct {
	RequestID           string `json:"request_id"`
	InputDataSize       string `json:"input_data_size"`
	ContentType         string `json:"content-type"`
	ServiceMethod       string `json:"service_method"`
	UserAgent           string `json:"user-agent"`
	RequestReceivedTime string `json:"request_arrival_time"`
	OrganizationID      string `json:"organization_id"`
	ServiceID           string `json:"service_id"`
	GroupID             string `json:"Group_id"`
	DaemonEndPoint      string `json:"Daemon_end_point"`
}

//Create a request Object and Publish this to a service end point
func PublishRequestStats(commonStat *CommonStats, inStream grpc.ServerStream) bool {
	request := createRequestStat(commonStat)
	if md, ok := metadata.FromIncomingContext(inStream.Context()); ok {
		request.setDataFromContext(md)
	}
	return Publish(request, config.GetString(config.MonitoringServiceEndpoint)+"/event")
}

func (request *RequestStats) setDataFromContext(md metadata.MD) {
	request.UserAgent = GetValue(md, "user-agent")
	request.ContentType = GetValue(md, "content-type")
	request.InputDataSize = strconv.FormatUint(GetSize(md), 10)
}

func createRequestStat(commonStat *CommonStats) *RequestStats {
	request := &RequestStats{
		RequestID:           commonStat.ID,
		GroupID:             commonStat.GroupID,
		DaemonEndPoint:      commonStat.DaemonEndPoint,
		OrganizationID:      commonStat.OrganizationID,
		ServiceID:           commonStat.ServiceID,
		RequestReceivedTime: commonStat.RequestReceivedTime,
	}
	return request
}
