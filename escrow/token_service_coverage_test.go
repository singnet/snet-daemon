package escrow

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tokenChannelServiceMock struct {
	paymentChannelFn          func(*PaymentChannelKey) (*PaymentChannelData, bool, error)
	startPaymentTransactionFn func(*Payment) (PaymentTransaction, error)
}

func (m *tokenChannelServiceMock) PaymentChannel(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
	if m.paymentChannelFn != nil {
		return m.paymentChannelFn(key)
	}
	return nil, false, nil
}

func (m *tokenChannelServiceMock) ListChannels() ([]*PaymentChannelData, error) { return nil, nil }

func (m *tokenChannelServiceMock) StartClaim(key *PaymentChannelKey, update ChannelUpdate) (Claim, error) {
	return nil, nil
}

func (m *tokenChannelServiceMock) ListClaims() ([]Claim, error) { return nil, nil }

func (m *tokenChannelServiceMock) StartPaymentTransaction(payment *Payment) (PaymentTransaction, error) {
	if m.startPaymentTransactionFn != nil {
		return m.startPaymentTransactionFn(payment)
	}
	return nil, nil
}

func (m *tokenChannelServiceMock) PaymentChannelFromBlockChain(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
	return nil, false, nil
}

type tokenPrePaidServiceMock struct {
	usage         map[string]*PrePaidData
	getErr        error
	updateErr     error
	updatedType   string
	updatedAmount *big.Int
}

func (m *tokenPrePaidServiceMock) GetUsage(key PrePaidDataKey) (*PrePaidData, bool, error) {
	if m.getErr != nil {
		return nil, false, m.getErr
	}
	if m.usage != nil {
		if d, ok := m.usage[key.UsageType]; ok {
			return d, true, nil
		}
	}
	return nil, false, nil
}

func (m *tokenPrePaidServiceMock) UpdateUsage(channelId *big.Int, amount *big.Int, typ string) error {
	m.updatedType = typ
	m.updatedAmount = amount
	return m.updateErr
}

type tokenServiceTokenManager struct {
	createErr error
}

func (m *tokenServiceTokenManager) CreateToken(key token.PayLoad, signer string) (token.CustomToken, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return "generated-token", nil
}

func (m *tokenServiceTokenManager) VerifyToken(t token.CustomToken, key token.PayLoad) (string, error) {
	return "", nil
}

type tokenServiceFixture struct {
	service      *TokenService
	channelMock  *tokenChannelServiceMock
	usageMock    *tokenPrePaidServiceMock
	tokenMock    *tokenServiceTokenManager
	senderKey    *ecdsa.PrivateKey
	senderAddr   common.Address
	receiverAddr common.Address
	svcMD        *blockchain.ServiceMetadata
}

func newTokenServiceFixture(t *testing.T) *tokenServiceFixture {
	t.Helper()
	config.Vip().Set(config.DaemonGroupName, "default_group")

	senderKey := GenerateTestPrivateKey()
	senderAddr := crypto.PubkeyToAddress(senderKey.PublicKey)
	receiverAddr := crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)

	svcMD, err := blockchain.InitServiceMetaDataFromJson([]byte(minimalControlServiceJSON))
	require.NoError(t, err)

	channelMock := &tokenChannelServiceMock{}
	usageMock := &tokenPrePaidServiceMock{}
	tokenMock := &tokenServiceTokenManager{}
	validator := &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
	service := NewTokenService(channelMock, usageMock, tokenMock, validator, svcMD)

	return &tokenServiceFixture{
		service:      service,
		channelMock:  channelMock,
		usageMock:    usageMock,
		tokenMock:    tokenMock,
		senderKey:    senderKey,
		senderAddr:   senderAddr,
		receiverAddr: receiverAddr,
		svcMD:        svcMD,
	}
}

func (f *tokenServiceFixture) channel() *PaymentChannelData {
	return &PaymentChannelData{
		ChannelID:        big.NewInt(1),
		Nonce:            big.NewInt(0),
		Sender:           f.senderAddr,
		Recipient:        f.receiverAddr,
		Signer:           f.receiverAddr,
		GroupID:          [32]byte{123},
		FullAmount:       big.NewInt(20),
		Expiration:       big.NewInt(1000),
		AuthorizedAmount: big.NewInt(10),
	}
}

func (f *tokenServiceFixture) signRequest(request *TokenRequest, key *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		f.svcMD.GetMpeAddress().Bytes(),
		bigIntToBytes(big.NewInt(1)),
		bigIntToBytes(big.NewInt(0).SetUint64(request.CurrentNonce)),
		bigIntToBytes(big.NewInt(0).SetUint64(request.SignedAmount)),
	}, nil)
	request.ClaimSignature = getSignature(message, key)
	message = bytes.Join([][]byte{
		request.ClaimSignature,
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	request.Signature = getSignature(message, key)
}

func TestNewTokenService(t *testing.T) {
	f := newTokenServiceFixture(t)
	require.NotNil(t, f.service)

	assert.NoError(t, f.service.allowedBlockNumberCheck(big.NewInt(99)))
	assert.Error(t, f.service.allowedBlockNumberCheck(big.NewInt(200)))
}

func TestTokenServiceGetPayment(t *testing.T) {
	f := newTokenServiceFixture(t)
	request := &TokenRequest{CurrentNonce: 3, SignedAmount: 11, ClaimSignature: []byte{1}}

	payment := f.service.getPayment(big.NewInt(1), big.NewInt(11), request)
	assert.Equal(t, big.NewInt(1), payment.ChannelID)
	assert.Equal(t, uint64(3), payment.ChannelNonce.Uint64())
	assert.Equal(t, big.NewInt(11), payment.Amount)
	assert.Equal(t, []byte{1}, payment.Signature)
	assert.Equal(t, f.svcMD.GetMpeAddress(), payment.MpeContractAddress)
}

func TestTokenServiceVerifySignature(t *testing.T) {
	f := newTokenServiceFixture(t)
	channel := f.channel()

	t.Run("incorrect signature", func(t *testing.T) {
		request := &TokenRequest{Signature: []byte{0x01}, ClaimSignature: []byte{0x02}, CurrentBlock: 99}
		_, err := f.service.verifySignature(request, channel)
		assert.EqualError(t, err, "incorrect signature")
	})

	t.Run("invalid signer", func(t *testing.T) {
		request := &TokenRequest{SignedAmount: 11, CurrentBlock: 99, CurrentNonce: 0}
		f.signRequest(request, GenerateTestPrivateKey())
		_, err := f.service.verifySignature(request, channel)
		assert.EqualError(t, err, "only channel signer/sender/receiver can get a Valid Token")
	})

	t.Run("expired block number", func(t *testing.T) {
		request := &TokenRequest{SignedAmount: 11, CurrentBlock: 200, CurrentNonce: 0}
		f.signRequest(request, f.senderKey)
		_, err := f.service.verifySignature(request, channel)
		assert.EqualError(t, err, "authentication failed as the signature passed has expired")
	})

	t.Run("success", func(t *testing.T) {
		request := &TokenRequest{SignedAmount: 11, CurrentBlock: 99, CurrentNonce: 0}
		f.signRequest(request, f.senderKey)
		signer, err := f.service.verifySignature(request, channel)
		assert.NoError(t, err)
		assert.Equal(t, f.senderAddr, *signer)
	})
}

func TestTokenServiceVerifySignatureAndSignedAmountEligibility(t *testing.T) {
	newRequest := func() *TokenRequest {
		return &TokenRequest{ChannelId: 1, SignedAmount: 11, CurrentNonce: 0, CurrentBlock: 99}
	}

	t.Run("channel not found", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return nil, false, nil
		}
		_, err := f.service.verifySignatureAndSignedAmountEligibility(big.NewInt(1), big.NewInt(11), newRequest())
		assert.EqualError(t, err, "channel is not found, channelId: 1")
	})

	t.Run("payment channel error", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return nil, true, errors.New("some error")
		}
		_, err := f.service.verifySignatureAndSignedAmountEligibility(big.NewInt(1), big.NewInt(11), newRequest())
		assert.EqualError(t, err, "error:some error was seen on retrieving details of channelID:1")
	})

	t.Run("signed amount greater than full amount", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return f.channel(), true, nil
		}
		_, err := f.service.verifySignatureAndSignedAmountEligibility(big.NewInt(1), big.NewInt(30), newRequest())
		assert.EqualError(t, err, "signed amount for token request cannot be greater than full amount in channel")
	})

	t.Run("signed amount less than authorized amount", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return f.channel(), true, nil
		}
		_, err := f.service.verifySignatureAndSignedAmountEligibility(big.NewInt(1), big.NewInt(3), newRequest())
		assert.EqualError(t, err, "signed amount for token request needs to be greater than last signed amount")
	})

	t.Run("success updates planned amount", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return f.channel(), true, nil
		}
		f.channelMock.startPaymentTransactionFn = func(*Payment) (PaymentTransaction, error) {
			return &paymentTransactionMock{}, nil
		}
		request := newRequest()
		f.signRequest(request, f.senderKey)

		signer, err := f.service.verifySignatureAndSignedAmountEligibility(big.NewInt(1), big.NewInt(11), request)
		assert.NoError(t, err)
		assert.Equal(t, f.senderAddr, *signer)
		assert.Equal(t, PLANNED_AMOUNT, f.usageMock.updatedType)
		assert.Equal(t, int64(1), f.usageMock.updatedAmount.Int64())
	})
}

func TestTokenServiceGetToken(t *testing.T) {
	newRequest := func() *TokenRequest {
		return &TokenRequest{ChannelId: 1, SignedAmount: 11, CurrentNonce: 0, CurrentBlock: 99}
	}

	t.Run("success", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return f.channel(), true, nil
		}
		f.channelMock.startPaymentTransactionFn = func(*Payment) (PaymentTransaction, error) {
			return &paymentTransactionMock{}, nil
		}
		f.usageMock.usage = map[string]*PrePaidData{
			USED_AMOUNT:    {Amount: big.NewInt(0)},
			PLANNED_AMOUNT: {Amount: big.NewInt(1)},
		}
		request := newRequest()
		f.signRequest(request, f.senderKey)

		reply, err := f.service.GetToken(context.Background(), request)
		assert.NoError(t, err)
		require.NotNil(t, reply)
		assert.Equal(t, uint64(1), reply.ChannelId)
		assert.Equal(t, "generated-token", reply.Token)
		assert.Equal(t, uint64(1), reply.PlannedAmount)
		assert.Equal(t, uint64(0), reply.UsedAmount)
	})

	t.Run("missing planned amount", func(t *testing.T) {
		f := newTokenServiceFixture(t)
		f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
			return f.channel(), true, nil
		}
		f.channelMock.startPaymentTransactionFn = func(*Payment) (PaymentTransaction, error) {
			return &paymentTransactionMock{}, nil
		}
		// no PLANNED_AMOUNT set
		f.usageMock.usage = map[string]*PrePaidData{USED_AMOUNT: {Amount: big.NewInt(0)}}
		request := newRequest()
		f.signRequest(request, f.senderKey)

		reply, err := f.service.GetToken(context.Background(), request)
		assert.Nil(t, reply)
		assert.EqualError(t, err, "unable to retrieve planned Amount <nil>")
	})
}
