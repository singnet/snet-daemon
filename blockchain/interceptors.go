package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (p Processor) jobValidationInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	jobAddressMd, ok := md[JobAddressHeader]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing snet-job-address")
	}

	jobAddressBytes := common.FromHex(jobAddressMd[0])

	jobSignatureMd, ok := md[JobSignatureHeader]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing snet-job-signature")
	}

	jobSignatureBytes := common.FromHex(jobSignatureMd[0])

	if !p.IsValidJobInvocation(jobAddressBytes, jobSignatureBytes) {
		return status.Errorf(codes.Unauthenticated, "job invocation not valid")
	}

	if err := handler(srv, ss); err != nil {
		return err
	}

	p.CompleteJob(jobAddressBytes, jobSignatureBytes)
	return nil
}

func noOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
