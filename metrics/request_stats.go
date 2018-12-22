package metrics

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"reflect"
)

type RequestStats struct {
	RequestID          string `json:"request_id"`
	InputDataSize      int    `json:"input_data_size"`
	ContentType        string `json:"content-type"`
	ServiceMethod      string `json:"service_method"`
	UserAgent          string `json:"user-agent"`
	RequestArrivalTime string `json:"request_arrival_time"`
	OrganizationID     string `json:"organization_id"`
	ServiceID          string `json:"service_id"`
	GroupID            string `json:"Group_id"`
	DaemonEndPoint     string `json:"Daemon_end_point"`
}

func PublishRequestStats(reqId string, grpId string, inStream grpc.ServerStream) *RequestStats {
	incomingContext := inStream.Context()
	request := createRequestStat(reqId, grpId)
	md, ok := metadata.FromIncomingContext(incomingContext)
	if ok {
		setDataFromContext(md, request)
	}
	request.ServiceMethod, _ = grpc.MethodFromServerStream(inStream)
	request.InputDataSize = getSize(inStream)
	json, _ := ConvertObjectToJSON(request)
	//Publish the request json created, loggin them for now
	log.WithField("Request Object ", json).Debug("Request Stats ")
	return request
}

func setDataFromContext(md metadata.MD, request *RequestStats) {
	request.UserAgent = GetValue(md, "user-agent")
	request.RequestArrivalTime = GetValue(md, "time")
	request.ContentType = GetValue(md, "content-type")
}

//ToDO
func getSize(T interface{}) int {
	v := reflect.TypeOf(T).Size()
	return int(v)
}

func createRequestStat(reqId string, grpId string) *RequestStats {
	request := &RequestStats{
		RequestID:      reqId,
		GroupID:        grpId,
		DaemonEndPoint: config.GetString(config.DaemonEndPoint),
		OrganizationID: config.GetString(config.OrganizationId),
		ServiceID:      config.GetString(config.ServiceId),
	}
	return request
}
