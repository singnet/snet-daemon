package escrow

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type incomeUnaryValidatorMockType struct {
	err error
}

func (v *incomeUnaryValidatorMockType) Validate(data *IncomeUnaryData) error {
	return v.err
}

func trainTestMetadata(channelID, channelNonce, amount int64, signature []byte) metadata.MD {
	md := metadata.New(map[string]string{})

	md.Set(handler.PaymentChannelIDHeader, strconv.FormatInt(channelID, 10))
	md.Set(handler.PaymentChannelNonceHeader, strconv.FormatInt(channelNonce, 10))
	md.Set(handler.PaymentChannelAmountHeader, strconv.FormatInt(amount, 10))
	md.Set(handler.PaymentChannelSignatureHeader, string(signature))

	return md
}

func trainTestChannel() *PaymentChannelData {
	return &PaymentChannelData{AuthorizedAmount: big.NewInt(12300)}
}

func newTrainStreamHandler(service PaymentChannelService, validator IncomeStreamValidator) *trainStreamPaymentHandler {
	return &trainStreamPaymentHandler{
		service:            service,
		mpeContractAddress: func() common.Address { return utils.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf") },
		currentBlock:       func() (*big.Int, error) { return big.NewInt(99), nil },
		incomeValidator:    validator,
	}
}

func newTrainUnaryHandler(service PaymentChannelService, validator IncomeUnaryValidator) *trainUnaryPaymentHandler {
	return &trainUnaryPaymentHandler{
		service:            service,
		mpeContractAddress: func() common.Address { return utils.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf") },
		currentBlock:       func() (*big.Int, error) { return big.NewInt(99), nil },
		incomeValidator:    validator,
	}
}

func newTrainStreamService(err error) PaymentChannelService {
	return &paymentChannelServiceMock{data: trainTestChannel(), err: err}
}

func TestTrainStreamPaymentHandlerType(t *testing.T) {
	h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
	assert.Equal(t, TrainPaymentType, h.Type())
}

func TestTrainStreamPaymentHandlerPayment(t *testing.T) {
	t.Run("valid payment", func(t *testing.T) {
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
		context := &handler.GrpcStreamContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1, 0x2})}

		payment, err := h.Payment(context)

		assert.Nil(t, err)
		assert.NotNil(t, payment)
	})

	t.Run("missing channel id", func(t *testing.T) {
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelIDHeader)

		payment, err := h.Payment(&handler.GrpcStreamContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing channel nonce", func(t *testing.T) {
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelNonceHeader)

		payment, err := h.Payment(&handler.GrpcStreamContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing channel amount", func(t *testing.T) {
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelAmountHeader)

		payment, err := h.Payment(&handler.GrpcStreamContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing signature", func(t *testing.T) {
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelSignatureHeader)

		payment, err := h.Payment(&handler.GrpcStreamContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-signature-bin\""), err)
		assert.Nil(t, payment)
	})

	t.Run("start transaction error", func(t *testing.T) {
		service := newTrainStreamService(NewPaymentError(FailedPrecondition, "another transaction in progress"))
		h := newTrainStreamHandler(service, &incomeValidatorMockType{})
		context := &handler.GrpcStreamContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1})}

		payment, err := h.Payment(context)

		assert.Equal(t, handler.NewGrpcError(codes.FailedPrecondition, "another transaction in progress"), err)
		assert.Nil(t, payment)
	})

	t.Run("incorrect income", func(t *testing.T) {
		incomeErr := NewPaymentError(Unauthenticated, "income does not equal to price")
		h := newTrainStreamHandler(newTrainStreamService(nil), &incomeValidatorMockType{err: incomeErr})
		context := &handler.GrpcStreamContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1})}

		payment, err := h.Payment(context)

		assert.Equal(t, handler.NewGrpcError(codes.Unauthenticated, "income does not equal to price"), err)
		assert.Nil(t, payment)
	})
}

func TestNewTrainStreamPaymentHandler(t *testing.T) {
	processor := blockchain.NewMockProcessor(true)
	h := NewTrainStreamPaymentHandler(newTrainStreamService(nil), processor, &incomeValidatorMockType{})

	require.NotNil(t, h)
	assert.Equal(t, TrainPaymentType, h.Type())
}

func TestTrainUnaryPaymentHandlerType(t *testing.T) {
	h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
	assert.Equal(t, TrainPaymentType, h.Type())
}

func TestTrainUnaryPaymentHandlerPayment(t *testing.T) {
	t.Run("valid payment", func(t *testing.T) {
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
		context := &handler.GrpcUnaryContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1, 0x2})}

		payment, err := h.Payment(context)

		assert.Nil(t, err)
		assert.NotNil(t, payment)
	})

	t.Run("missing channel id", func(t *testing.T) {
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelIDHeader)

		payment, err := h.Payment(&handler.GrpcUnaryContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing channel nonce", func(t *testing.T) {
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelNonceHeader)

		payment, err := h.Payment(&handler.GrpcUnaryContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing channel amount", func(t *testing.T) {
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelAmountHeader)

		payment, err := h.Payment(&handler.GrpcUnaryContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
		assert.Nil(t, payment)
	})

	t.Run("missing signature", func(t *testing.T) {
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{})
		md := trainTestMetadata(42, 3, 12345, []byte{0x1})
		delete(md, handler.PaymentChannelSignatureHeader)

		payment, err := h.Payment(&handler.GrpcUnaryContext{MD: md})

		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-signature-bin\""), err)
		assert.Nil(t, payment)
	})

	t.Run("start transaction error", func(t *testing.T) {
		service := newTrainStreamService(NewPaymentError(FailedPrecondition, "another transaction in progress"))
		h := newTrainUnaryHandler(service, &incomeUnaryValidatorMockType{})
		context := &handler.GrpcUnaryContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1})}

		payment, err := h.Payment(context)

		assert.Equal(t, handler.NewGrpcError(codes.FailedPrecondition, "another transaction in progress"), err)
		assert.Nil(t, payment)
	})

	t.Run("incorrect income", func(t *testing.T) {
		incomeErr := NewPaymentError(Unauthenticated, "income does not equal to price")
		h := newTrainUnaryHandler(newTrainStreamService(nil), &incomeUnaryValidatorMockType{err: incomeErr})
		context := &handler.GrpcUnaryContext{MD: trainTestMetadata(42, 3, 12345, []byte{0x1})}

		payment, err := h.Payment(context)

		assert.Equal(t, handler.NewGrpcError(codes.Unauthenticated, "income does not equal to price"), err)
		assert.Nil(t, payment)
	})
}

func TestNewTrainUnaryPaymentHandler(t *testing.T) {
	processor := blockchain.NewMockProcessor(true)
	h := NewTrainUnaryPaymentHandler(newTrainStreamService(nil), processor, &incomeUnaryValidatorMockType{})

	require.NotNil(t, h)
	assert.Equal(t, TrainPaymentType, h.Type())
}

func newTestPaymentTransaction() *paymentTransaction {
	memStorage := storage.NewMemStorage()
	return &paymentTransaction{
		payment: Payment{
			ChannelID: big.NewInt(42),
			Amount:    big.NewInt(12345),
			Signature: []byte{0x1},
		},
		channel: &PaymentChannelData{
			ChannelID:  big.NewInt(42),
			Nonce:      big.NewInt(3),
			Sender:     common.HexToAddress("0x1"),
			Recipient:  common.HexToAddress("0x2"),
			FullAmount: big.NewInt(12300),
			Expiration: big.NewInt(100),
			Signer:     common.HexToAddress("0x3"),
			GroupID:    [32]byte{},
		},
		service: &lockingPaymentChannelService{
			storage: NewPaymentChannelStorage(memStorage),
		},
		lock: &lockType{name: "test", locker: &etcdLocker{storage: memStorage}},
	}
}

func TestTrainStreamPaymentHandlerComplete(t *testing.T) {
	config.Vip().Set(config.MeteringEnabled, false)
	h := newTrainStreamHandler(nil, &incomeValidatorMockType{})

	err := h.Complete(newTestPaymentTransaction())

	assert.Nil(t, err)
}

func TestTrainStreamPaymentHandlerCompleteAfterError(t *testing.T) {
	h := newTrainStreamHandler(nil, &incomeValidatorMockType{})

	err := h.CompleteAfterError(newTestPaymentTransaction(), nil)

	assert.Nil(t, err)
}

func TestTrainUnaryPaymentHandlerComplete(t *testing.T) {
	config.Vip().Set(config.MeteringEnabled, false)
	h := newTrainUnaryHandler(nil, &incomeUnaryValidatorMockType{})

	err := h.Complete(newTestPaymentTransaction())

	assert.Nil(t, err)
}

func TestTrainUnaryPaymentHandlerCompleteAfterError(t *testing.T) {
	h := newTrainUnaryHandler(nil, &incomeUnaryValidatorMockType{})

	err := h.CompleteAfterError(newTestPaymentTransaction(), nil)

	assert.Nil(t, err)
}
