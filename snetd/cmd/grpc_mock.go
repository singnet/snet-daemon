//go:generate protoc grpc_mock.proto --go_out=plugins=grpc:.

package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
)



var (

	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	jsonDBFile = flag.String("json_db_file", "", "A json file containing a list of features")
	port       = flag.Int("port", 8086, "The server port")
)
type ServiceMock struct {
	output *Output
	err    error
}

var ch = make(chan int)
var sigChan = make(chan os.Signal, 1)

func (service *ServiceMock) LongCall(context context.Context, input *Input) (output *Output, err error) {
	<-sigChan
	fmt.Printf("Call to service reached ... Service Provider .....")
	return service.output, service.err
}



