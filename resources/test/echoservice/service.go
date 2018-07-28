package echoservice

//go:generate protoc --go_out=plugins=grpc:. api.proto

// "protoc" should be installed by your system's package manager
// protoc-gen-go is installed with:
//
//   go get -u -x github.com/golang/protobuf/protoc-gen-go
//
// The grpc plugin is installed with:
//
//    go get -u google.golang.org/grpc
