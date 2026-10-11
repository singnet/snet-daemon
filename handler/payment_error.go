package handler

import (
	"github.com/singnet/snet-daemon/v6/errs"
	"google.golang.org/grpc/codes"
)

// PaymentErrorToGRPCError maps payment errors to the public gRPC status codes.
func PaymentErrorToGRPCError(err error) *GrpcError {
	if err == nil {
		return nil
	}

	paymentErr, ok := err.(*errs.PaymentError)
	if !ok {
		return NewGrpcErrorf(codes.Internal, "internal error: %v", err)
	}

	var grpcCode codes.Code
	switch paymentErr.Code {
	case errs.Internal:
		grpcCode = codes.Internal
	case errs.Unauthenticated:
		grpcCode = codes.Unauthenticated
	case errs.FailedPrecondition:
		grpcCode = codes.FailedPrecondition
	case errs.IncorrectNonce:
		grpcCode = IncorrectNonce
	default:
		grpcCode = codes.Internal
	}

	return NewGrpcError(grpcCode, paymentErr.Message)
}
