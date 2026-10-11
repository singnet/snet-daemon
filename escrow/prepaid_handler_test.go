package escrow

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/pricing"
	"github.com/singnet/snet-daemon/v6/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const minimalPrepaidServiceJSON = `{"version":1,"display_name":"test","encoding":"grpc","service_type":"grpc","mpe_address":"0x39ee715b50e78a920120c1ded58b1a47f571ab75","groups":[{"group_name":"default_group","pricing":[{"price_model":"fixed_price","default":true,"price_in_cogs":1}]}]}`

type mockTokenManager struct {
	verifyErr error
	address   string
}

func (m *mockTokenManager) CreateToken(key token.PayLoad, signer string) (token.CustomToken, error) {
	return nil, nil
}

func (m *mockTokenManager) VerifyToken(t token.CustomToken, key token.PayLoad) (string, error) {
	if m.verifyErr != nil {
		return "", m.verifyErr
	}
	return m.address, nil
}

type mockPrePaidService struct {
	updateErr      error
	updatedChannel *big.Int
	updatedAmount  *big.Int
	updatedType    string
}

func (m *mockPrePaidService) GetUsage(key PrePaidDataKey) (*PrePaidData, bool, error) {
	return nil, false, nil
}

func (m *mockPrePaidService) UpdateUsage(channelId *big.Int, revisedAmount *big.Int, updateUsageType string) error {
	m.updatedChannel = channelId
	m.updatedAmount = revisedAmount
	m.updatedType = updateUsageType
	return m.updateErr
}

type fixedPriceMockType struct{ price *big.Int }

func (p *fixedPriceMockType) GetPrice(ctx *handler.GrpcStreamContext) (*big.Int, error) {
	return p.price, nil
}
func (p *fixedPriceMockType) GetPriceType() string { return pricing.FIXED_PRICING }

type errorPriceMockType struct{}

func (p *errorPriceMockType) GetPrice(ctx *handler.GrpcStreamContext) (*big.Int, error) {
	return nil, fmt.Errorf("price error")
}
func (p *errorPriceMockType) GetPriceType() string { return pricing.FIXED_PRICING }

func prepaidPricingStrategy(t *testing.T, priceType pricing.PriceType) *pricing.PricingStrategy {
	t.Helper()
	svcMD, err := blockchain.InitServiceMetaDataFromJson([]byte(minimalPrepaidServiceJSON))
	require.NoError(t, err)
	strategy, err := pricing.InitPricingStrategy(svcMD)
	require.NoError(t, err)
	strategy.AddPricingTypes(priceType)
	return strategy
}

func prepaidTestOrgMetadata(t *testing.T) *blockchain.OrganizationMetaData {
	t.Helper()
	config.Vip().Set(config.DaemonGroupName, "default_group")
	md, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	require.NoError(t, err)
	return md
}

func prepaidContext(t *testing.T) *handler.GrpcStreamContext {
	t.Helper()
	md := metadata.New(map[string]string{})
	md.Set(handler.PaymentChannelIDHeader, "42")
	md.Set(handler.PrePaidAuthTokenHeader, "some-token")
	return &handler.GrpcStreamContext{MD: md, Info: &grpc.StreamServerInfo{FullMethod: "/svc/method"}}
}

func TestPrePaidPaymentHandlerType(t *testing.T) {
	h := &PrePaidPaymentHandler{}
	assert.Equal(t, PrePaidPaymentType, h.Type())
}

func TestPrePaidPaymentValidatorValidate(t *testing.T) {
	t.Run("verify token error", func(t *testing.T) {
		validator := NewPrePaidPaymentValidator(nil, &mockTokenManager{verifyErr: errors.New("invalid token")})
		addr, err := validator.Validate(&PrePaidPayment{AuthToken: "bad", ChannelID: big.NewInt(1)})
		assert.Equal(t, common.Address{}, addr)
		assert.EqualError(t, err, "invalid token")
	})

	t.Run("success returns user address", func(t *testing.T) {
		validator := NewPrePaidPaymentValidator(nil, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})
		addr, err := validator.Validate(&PrePaidPayment{AuthToken: "good", ChannelID: big.NewInt(1)})
		assert.NoError(t, err)
		assert.Equal(t, common.HexToAddress("0x671276c61943A35D5F230d076bDFd91B0c47bF09"), addr)
	})
}

func TestPrePaidPaymentHandlerPayment(t *testing.T) {
	newHandler := func(t *testing.T, service *mockPrePaidService, priceType pricing.PriceType, tokenManager *mockTokenManager) *PrePaidPaymentHandler {
		t.Helper()
		return &PrePaidPaymentHandler{
			service:                 service,
			orgMetadata:             prepaidTestOrgMetadata(t),
			PrePaidPaymentValidator: NewPrePaidPaymentValidator(prepaidPricingStrategy(t, priceType), tokenManager),
		}
	}

	t.Run("missing channel id", func(t *testing.T) {
		h := newHandler(t, &mockPrePaidService{}, &fixedPriceMockType{price: big.NewInt(1)}, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})
		ctx := prepaidContext(t)
		delete(ctx.MD, handler.PaymentChannelIDHeader)

		transaction, err := h.Payment(ctx)
		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
		assert.Nil(t, transaction)
	})

	t.Run("missing auth token", func(t *testing.T) {
		h := newHandler(t, &mockPrePaidService{}, &fixedPriceMockType{price: big.NewInt(1)}, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})
		ctx := prepaidContext(t)
		delete(ctx.MD, handler.PrePaidAuthTokenHeader)

		transaction, err := h.Payment(ctx)
		assert.Equal(t, handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-prepaid-auth-token-bin\""), err)
		assert.Nil(t, transaction)
	})

	t.Run("price error", func(t *testing.T) {
		h := newHandler(t, &mockPrePaidService{}, &errorPriceMockType{}, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})

		transaction, err := h.Payment(prepaidContext(t))
		require.Error(t, err)
		assert.Nil(t, transaction)
	})

	t.Run("validate error", func(t *testing.T) {
		h := newHandler(t, &mockPrePaidService{}, &fixedPriceMockType{price: big.NewInt(1)}, &mockTokenManager{verifyErr: errors.New("invalid token")})

		transaction, err := h.Payment(prepaidContext(t))
		require.Error(t, err)
		assert.Nil(t, transaction)
	})

	t.Run("update usage error", func(t *testing.T) {
		h := newHandler(t, &mockPrePaidService{updateErr: errors.New("usage error")}, &fixedPriceMockType{price: big.NewInt(1)}, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})

		transaction, err := h.Payment(prepaidContext(t))
		require.Error(t, err)
		assert.Nil(t, transaction)
	})

	t.Run("success", func(t *testing.T) {
		service := &mockPrePaidService{}
		h := newHandler(t, service, &fixedPriceMockType{price: big.NewInt(7)}, &mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"})

		transaction, err := h.Payment(prepaidContext(t))
		assert.Nil(t, err)
		require.NotNil(t, transaction)

		assert.Equal(t, int64(42), service.updatedChannel.Int64())
		assert.Equal(t, int64(7), service.updatedAmount.Int64())
		assert.Equal(t, USED_AMOUNT, service.updatedType)

		impl := transaction.(*prePaidTransactionImpl)
		assert.Equal(t, int64(42), impl.ChannelId().Int64())
		assert.Equal(t, int64(7), impl.Price().Int64())
	})
}

func TestPrePaidPaymentHandlerComplete(t *testing.T) {
	h := &PrePaidPaymentHandler{}
	err := h.Complete(&prePaidTransactionImpl{price: big.NewInt(5), channelId: big.NewInt(1)})
	assert.Nil(t, err)
}

func TestPrePaidPaymentHandlerCompleteAfterError(t *testing.T) {
	t.Run("refund succeeds", func(t *testing.T) {
		service := &mockPrePaidService{}
		h := &PrePaidPaymentHandler{service: service}

		err := h.CompleteAfterError(&prePaidTransactionImpl{price: big.NewInt(5), channelId: big.NewInt(1)}, errors.New("service error"))
		assert.Nil(t, err)
		assert.Equal(t, REFUND_AMOUNT, service.updatedType)
		assert.Equal(t, int64(5), service.updatedAmount.Int64())
	})

	t.Run("refund fails", func(t *testing.T) {
		service := &mockPrePaidService{updateErr: errors.New("refund error")}
		h := &PrePaidPaymentHandler{service: service}

		err := h.CompleteAfterError(&prePaidTransactionImpl{price: big.NewInt(5), channelId: big.NewInt(1)}, errors.New("service error"))
		require.Error(t, err)
	})
}

func TestNewPrePaidPaymentHandler(t *testing.T) {
	h := NewPrePaidPaymentHandler(
		&mockPrePaidService{},
		prepaidTestOrgMetadata(t),
		&blockchain.ServiceMetadata{MpeAddress: controlTestMpeAddress},
		prepaidPricingStrategy(t, &fixedPriceMockType{price: big.NewInt(1)}),
		&mockTokenManager{address: "0x671276c61943A35D5F230d076bDFd91B0c47bF09"},
	)
	require.NotNil(t, h)
	assert.Equal(t, PrePaidPaymentType, h.Type())
}
