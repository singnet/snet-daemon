package escrow

import (
	"errors"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestPaymentChannelPaymentHandlerType(t *testing.T) {
	h := &paymentChannelPaymentHandler{}
	assert.Equal(t, EscrowPaymentType, h.Type())
}

func TestNewPaymentHandler(t *testing.T) {
	h := NewPaymentHandler(&paymentChannelServiceMock{}, blockchain.NewMockProcessor(true), &incomeValidatorMockType{})
	assert.NotNil(t, h)
	assert.Equal(t, EscrowPaymentType, h.Type())
}

func TestPaymentChannelPaymentHandlerComplete(t *testing.T) {
	h := &paymentChannelPaymentHandler{}
	err := h.Complete(newTestPaymentTransaction())
	assert.Nil(t, err)
}

func TestPaymentChannelPaymentHandlerCompleteAfterError(t *testing.T) {
	h := &paymentChannelPaymentHandler{}
	err := h.CompleteAfterError(newTestPaymentTransaction(), nil)
	assert.Nil(t, err)
}

func TestPaymentErrorToGrpcError(t *testing.T) {
	assert.Nil(t, paymentErrorToGrpcError(nil))

	t.Run("plain error maps to internal", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(errors.New("plain error"))
		assert.Equal(t, codes.Internal, grpcErr.Status.Code())
	})

	t.Run("internal", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(NewPaymentError(Internal, "internal"))
		assert.Equal(t, codes.Internal, grpcErr.Status.Code())
	})

	t.Run("unauthenticated", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(NewPaymentError(Unauthenticated, "unauth"))
		assert.Equal(t, codes.Unauthenticated, grpcErr.Status.Code())
	})

	t.Run("failed precondition", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(NewPaymentError(FailedPrecondition, "failed"))
		assert.Equal(t, codes.FailedPrecondition, grpcErr.Status.Code())
	})

	t.Run("incorrect nonce", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(NewPaymentError(IncorrectNonce, "nonce"))
		assert.Equal(t, handler.IncorrectNonce, grpcErr.Status.Code())
	})

	t.Run("unknown code maps to internal", func(t *testing.T) {
		grpcErr := paymentErrorToGrpcError(NewPaymentError(PaymentErrorCode(999), "unknown"))
		assert.Equal(t, codes.Internal, grpcErr.Status.Code())
	})
}
