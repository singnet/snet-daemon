//go:generate protoc test_service.proto --go_out=plugins=grpc:.

package cmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/singnet/snet-daemon/config"

	"os"

	"net"


	"google.golang.org/grpc"

)


var (

	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	jsonDBFile = flag.String("json_db_file", "", "A json file containing a list of features")
	port       = flag.Int("port", 8086, "The server port")
)
type ServiceMock struct {

}

var ch = make(chan int)
var sigChan = make(chan os.Signal, 1)

func (service *ServiceMock) LongCall(context context.Context, input *TestMessage) (output *TestMessage, err error) {
	<-sigChan
	fmt.Printf("Call to service reached ... Service Provider .....")
	return &TestMessage{MessageString:"Hello from Service"}, nil
}

func StartMockService() {
	go func() {
		flag.Parse()
		lis, err := net.Listen("tcp", config.GetString(config.PassthroughEndpointKey))
		if err != nil {
			fmt.Sprintf("failed to listen: %v", err)
		}
		var opts []grpc.ServerOption

		fmt.Printf("Starting Service.....\n")
		grpcServer := grpc.NewServer(opts...)
		RegisterMockServiceServer(grpcServer, &ServiceMock{})
		ch <- 0
		grpcServer.Serve(lis)

	}()
}



