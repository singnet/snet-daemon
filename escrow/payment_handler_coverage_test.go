package escrow

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
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
	originalMetering := config.GetBool(config.MeteringEnabled)
	config.Vip().Set(config.MeteringEnabled, true)
	t.Cleanup(func() { config.Vip().Set(config.MeteringEnabled, originalMetering) })
	called := make(chan struct{})
	h := &paymentChannelPaymentHandler{currentBlock: func() (*big.Int, error) {
		close(called)
		return nil, errors.New("test block unavailable")
	}}
	err := h.Complete(newTestPaymentTransaction())
	assert.Nil(t, err)
	// Wait for the background publisher before other tests change global config.
	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("Complete did not invoke the stats publisher")
	}
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
