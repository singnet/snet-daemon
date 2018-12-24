package metrics

import (
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strconv"
	"time"
)

//Request stats that will be captured
type RequestStats struct {
	RequestID          string `json:"request_id"`
	InputDataSize      string `json:"input_data_size"`
	ContentType        string `json:"content-type"`
	ServiceMethod      string `json:"service_method"`
	UserAgent          string `json:"user-agent"`
	RequestArrivalTime string `json:"request_arrival_time"`
	OrganizationID     string `json:"organization_id"`
	ServiceID          string `json:"service_id"`
	GroupID            string `json:"Group_id"`
	DaemonEndPoint     string `json:"Daemon_end_point"`
}

//Create a request Object and Publish this to a service end point
func PublishRequestStats(reqId string, grpId string, arrivalTime time.Time, inStream grpc.ServerStream) bool {
	request := createRequestStat(reqId, grpId, arrivalTime)
	setDataFromInStream(inStream, request)
	return Publish(request, config.GetString(config.MonitoringServiceEndpoint))
}

func setDataFromInStream(inStream grpc.ServerStream, request *RequestStats) {
	request.ServiceMethod, _ = grpc.MethodFromServerStream(inStream)
	if md, ok := metadata.FromIncomingContext(inStream.Context()); ok {
		setDataFromContext(md, request)
	}
}

func setDataFromContext(md metadata.MD, request *RequestStats) {
	request.UserAgent = GetValue(md, "user-agent")
	request.ContentType = GetValue(md, "content-type")
	//todo
	request.InputDataSize = strconv.FormatUint(GetSize(md), 10)
}

func createRequestStat(reqId string, grpId string, time time.Time) *RequestStats {
	request := &RequestStats{
		RequestID:          reqId,
		GroupID:            grpId,
		DaemonEndPoint:     config.GetString(config.DaemonEndPoint),
		OrganizationID:     config.GetString(config.OrganizationId),
		ServiceID:          config.GetString(config.ServiceId),
		RequestArrivalTime: time.String(),
	}
	return request
}
