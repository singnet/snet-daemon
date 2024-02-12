package fix_proto

import "os"

func init() {
	// fixing the panic due to the fact that the libraries have the same proto files by name:
	// github.com/ipfs/go-cid conflicts with etcd library
	// https://protobuf.dev/reference/go/faq/#fix-namespace-conflict
	os.Setenv("GOLANG_PROTOBUF_REGISTRATION_CONFLICT", "warn")
}
