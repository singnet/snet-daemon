//go:generate protoc -I . ./state_service.proto --go_out=plugins=grpc:.
package escrow
