package handler

import (
	"errors"
	"testing"

	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPaymentErrorToGRPCError(t *testing.T) {
	require.Nil(t, PaymentErrorToGRPCError(nil))
	for _, tc := range []struct {
		name    string
		err     error
		code    codes.Code
		message string
	}{
		{"internal", errs.NewPaymentError(errs.Internal, "storage failed"), codes.Internal, "storage failed"},
		{"unauthenticated", errs.NewPaymentError(errs.Unauthenticated, "invalid signer"), codes.Unauthenticated, "invalid signer"},
		{"precondition", errs.NewPaymentError(errs.FailedPrecondition, "channel locked"), codes.FailedPrecondition, "channel locked"},
		{"nonce", errs.NewPaymentError(errs.IncorrectNonce, "incorrect nonce"), IncorrectNonce, "incorrect nonce"},
		{"unknown payment code", errs.NewPaymentError(999, "unexpected error"), codes.Internal, "unexpected error"},
		{"ordinary error", errors.New("backend failed"), codes.Internal, "internal error: backend failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := PaymentErrorToGRPCError(tc.err)
			require.Equal(t, tc.code, status.Code(result.Err()))
			require.Equal(t, tc.message, status.Convert(result.Err()).Message())
		})
	}
}
