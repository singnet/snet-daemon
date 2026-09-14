package escrow

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const controlTestMpeAddress = "0x39ee715b50e78a920120c1ded58b1a47f571ab75"

const minimalControlServiceJSON = `{"version":1,"display_name":"test","encoding":"grpc","service_type":"grpc","mpe_address":"` + controlTestMpeAddress + `","groups":[{"group_name":"default_group","pricing":[{"price_model":"fixed_price","default":true,"price_in_cogs":1}]}]}`

type controlServiceChannelMock struct {
	paymentChannelFn          func(*PaymentChannelKey) (*PaymentChannelData, bool, error)
	listChannelsFn            func() ([]*PaymentChannelData, error)
	listClaimsFn              func() ([]Claim, error)
	startClaimFn              func(*PaymentChannelKey, ChannelUpdate) (Claim, error)
	paymentChannelFromChainFn func(*PaymentChannelKey) (*PaymentChannelData, bool, error)
}

func (m *controlServiceChannelMock) PaymentChannel(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
	if m.paymentChannelFn != nil {
		return m.paymentChannelFn(key)
	}
	return nil, false, nil
}

func (m *controlServiceChannelMock) ListChannels() ([]*PaymentChannelData, error) {
	if m.listChannelsFn != nil {
		return m.listChannelsFn()
	}
	return nil, nil
}

func (m *controlServiceChannelMock) StartClaim(key *PaymentChannelKey, update ChannelUpdate) (Claim, error) {
	if m.startClaimFn != nil {
		return m.startClaimFn(key, update)
	}
	return nil, nil
}

func (m *controlServiceChannelMock) ListClaims() ([]Claim, error) {
	if m.listClaimsFn != nil {
		return m.listClaimsFn()
	}
	return nil, nil
}

func (m *controlServiceChannelMock) StartPaymentTransaction(payment *Payment) (PaymentTransaction, error) {
	return nil, nil
}

func (m *controlServiceChannelMock) PaymentChannelFromBlockChain(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
	if m.paymentChannelFromChainFn != nil {
		return m.paymentChannelFromChainFn(key)
	}
	return nil, false, nil
}

type mockClaim struct {
	payment   *Payment
	finishErr error
	finished  bool
}

func (c *mockClaim) Payment() *Payment { return c.payment }
func (c *mockClaim) Finish() error {
	c.finished = true
	return c.finishErr
}

type controlServiceFixture struct {
	service     *ProviderControlService
	channelMock *controlServiceChannelMock
	signerKey   *ecdsa.PrivateKey
	signerAddr  common.Address
	svcMD       *blockchain.ServiceMetadata
}

func newControlServiceFixture(t *testing.T) *controlServiceFixture {
	t.Helper()

	config.Vip().Set(config.DaemonGroupName, "default_group")

	signerKey := GenerateTestPrivateKey()
	signerAddr := crypto.PubkeyToAddress(signerKey.PublicKey)

	orgJSON := strings.Replace(testJsonOrgGroupData, "0x671276c61943A35D5F230d076bDFd91B0c47bF09", signerAddr.Hex(), -1)
	orgMD, err := blockchain.InitOrganizationMetaDataFromJson([]byte(orgJSON))
	require.NoError(t, err)

	svcMD, err := blockchain.InitServiceMetaDataFromJson([]byte(minimalControlServiceJSON))
	require.NoError(t, err)

	channelMock := &controlServiceChannelMock{}
	service := NewProviderControlService(blockchain.NewMockProcessor(true), channelMock, svcMD, orgMD)

	return &controlServiceFixture{
		service:     service,
		channelMock: channelMock,
		signerKey:   signerKey,
		signerAddr:  signerAddr,
		svcMD:       svcMD,
	}
}

func (f *controlServiceFixture) signListRequest(prefix string, currentBlock uint64) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefix),
		f.svcMD.GetMpeAddress().Bytes(),
		math.U256Bytes(big.NewInt(int64(currentBlock))),
	}, nil)
	return getSignature(message, f.signerKey)
}

func TestListChannels(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listChannelsFn = func() ([]*PaymentChannelData, error) {
		return []*PaymentChannelData{
			{ChannelID: big.NewInt(1), Nonce: big.NewInt(0), AuthorizedAmount: big.NewInt(0), Expiration: big.NewInt(100)},
			{ChannelID: big.NewInt(2), Nonce: big.NewInt(1), AuthorizedAmount: big.NewInt(50), Expiration: big.NewInt(200)},
		}, nil
	}

	reply, err := f.service.listChannels()
	assert.NoError(t, err)
	require.Len(t, reply.Payments, 1)
	assert.Equal(t, int64(2), bytesToBigInt(reply.Payments[0].ChannelId).Int64())
}

func TestListChannelsError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listChannelsFn = func() ([]*PaymentChannelData, error) {
		return nil, errors.New("storage error")
	}

	_, err := f.service.listChannels()
	assert.EqualError(t, err, "storage error")
}

func TestListClaims(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{Expiration: big.NewInt(300)}, true, nil
	}
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return []Claim{
			&mockClaim{payment: &Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(0), Amount: big.NewInt(10)}},                       // nil signature -> skipped
			&mockClaim{payment: &Payment{ChannelID: big.NewInt(2), ChannelNonce: big.NewInt(0), Amount: big.NewInt(0), Signature: []byte{1}}},  // zero amount -> skipped
			&mockClaim{payment: &Payment{ChannelID: big.NewInt(3), ChannelNonce: big.NewInt(1), Amount: big.NewInt(20), Signature: []byte{2}}}, // valid
		}, nil
	}

	reply, err := f.service.listClaims()
	assert.NoError(t, err)
	require.Len(t, reply.Payments, 1)
	assert.Equal(t, int64(3), bytesToBigInt(reply.Payments[0].ChannelId).Int64())
}

func TestListClaimsError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return nil, errors.New("claim error")
	}

	_, err := f.service.listClaims()
	assert.EqualError(t, err, "claim error")
}

func TestListClaimsPaymentChannelError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return []Claim{
			&mockClaim{payment: &Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(0), Amount: big.NewInt(10), Signature: []byte{1}}},
		}, nil
	}
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return nil, false, errors.New("channel error")
	}

	reply, err := f.service.listClaims()
	assert.NoError(t, err)
	assert.Empty(t, reply.Payments)
}

func TestBeginClaimOnChannelNotFound(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return nil, false, nil
	}

	_, err := f.service.beginClaimOnChannel(big.NewInt(42))
	assert.EqualError(t, err, "channel Id 42 was not found on blockchain or storage")
}

func TestBeginClaimOnChannelZeroAmount(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{AuthorizedAmount: big.NewInt(0)}, true, nil
	}

	_, err := f.service.beginClaimOnChannel(big.NewInt(42))
	assert.EqualError(t, err, "authorized amount is zero , hence nothing to claim on the channel Id: 42")
}

func TestBeginClaimOnChannelStartClaimError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{AuthorizedAmount: big.NewInt(100)}, true, nil
	}
	f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
		return nil, errors.New("claim failed")
	}

	_, err := f.service.beginClaimOnChannel(big.NewInt(42))
	assert.EqualError(t, err, "claim failed")
}

func TestBeginClaimOnChannelSuccess(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{AuthorizedAmount: big.NewInt(100)}, true, nil
	}
	f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
		return &mockClaim{payment: &Payment{ChannelNonce: big.NewInt(5), Amount: big.NewInt(100), Signature: []byte{9}}}, nil
	}

	reply, err := f.service.beginClaimOnChannel(big.NewInt(42))
	assert.NoError(t, err)
	assert.Equal(t, int64(42), bytesToBigInt(reply.ChannelId).Int64())
	assert.Equal(t, int64(5), bytesToBigInt(reply.ChannelNonce).Int64())
	assert.Equal(t, []byte{9}, reply.Signature)
	assert.Equal(t, int64(100), bytesToBigInt(reply.SignedAmount).Int64())
}

func TestGetBytesOfChannelIds(t *testing.T) {
	req := &StartMultipleClaimRequest{ChannelIds: []uint64{2, 1, 3}}
	got := getBytesOfChannelIds(req)
	want := bytes.Join([][]byte{
		bigIntToBytes(big.NewInt(1)),
		bigIntToBytes(big.NewInt(2)),
		bigIntToBytes(big.NewInt(3)),
	}, nil)
	assert.Equal(t, want, got)
}

func TestRemoveClaimedPayments(t *testing.T) {
	f := newControlServiceFixture(t)
	claimDone := &mockClaim{payment: &Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(1)}}
	claimPending := &mockClaim{payment: &Payment{ChannelID: big.NewInt(2), ChannelNonce: big.NewInt(5)}}
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return []Claim{claimDone, claimPending}, nil
	}
	f.channelMock.paymentChannelFromChainFn = func(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
		if key.ID.Int64() == 1 {
			return &PaymentChannelData{Nonce: big.NewInt(5)}, true, nil // blockchain nonce > payment nonce -> finish
		}
		return &PaymentChannelData{Nonce: big.NewInt(3)}, true, nil // blockchain nonce < payment nonce -> no finish
	}

	err := f.service.removeClaimedPayments()
	assert.NoError(t, err)
	assert.True(t, claimDone.finished)
	assert.False(t, claimPending.finished)
}

func TestRemoveClaimedPaymentsListError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return nil, errors.New("claim error")
	}

	err := f.service.removeClaimedPayments()
	assert.EqualError(t, err, "error in retrieving claims")
}

func TestRemoveClaimedPaymentsBlockchainError(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) {
		return []Claim{&mockClaim{payment: &Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(1)}}}, nil
	}
	f.channelMock.paymentChannelFromChainFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return nil, false, errors.New("blockchain error")
	}

	err := f.service.removeClaimedPayments()
	assert.EqualError(t, err, "blockchain error")
}

func TestVerifySigner(t *testing.T) {
	f := newControlServiceFixture(t)
	message := []byte("hello")
	signature := getSignature(message, f.signerKey)

	assert.NoError(t, f.service.verifySigner(message, signature))
}

func TestVerifySignerWrongSigner(t *testing.T) {
	f := newControlServiceFixture(t)
	message := []byte("hello")
	signature := getSignature(message, GenerateTestPrivateKey())

	assert.Error(t, f.service.verifySigner(message, signature))
}

func TestVerifySignerInvalidSignature(t *testing.T) {
	f := newControlServiceFixture(t)
	assert.Error(t, f.service.verifySigner([]byte("hello"), []byte{0x01, 0x02}))
}

func TestGetListUnclaimed(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listChannelsFn = func() ([]*PaymentChannelData, error) {
		return []*PaymentChannelData{
			{ChannelID: big.NewInt(1), Nonce: big.NewInt(0), AuthorizedAmount: big.NewInt(10), Expiration: big.NewInt(100)},
		}, nil
	}

	req := &GetPaymentsListRequest{MpeAddress: f.svcMD.MpeAddress, CurrentBlock: blockchain.MockedCurrentBlock}
	req.Signature = f.signListRequest("__list_unclaimed", req.CurrentBlock)

	reply, err := f.service.GetListUnclaimed(context.Background(), req)
	assert.NoError(t, err)
	require.Len(t, reply.Payments, 1)
}

func TestGetListUnclaimedBadMpe(t *testing.T) {
	f := newControlServiceFixture(t)
	req := &GetPaymentsListRequest{MpeAddress: "0x0000000000000000000000000000000000000000", CurrentBlock: blockchain.MockedCurrentBlock}

	_, err := f.service.GetListUnclaimed(context.Background(), req)
	assert.Error(t, err)
}

func TestGetListUnclaimedBadBlock(t *testing.T) {
	f := newControlServiceFixture(t)
	req := &GetPaymentsListRequest{MpeAddress: f.svcMD.MpeAddress, CurrentBlock: blockchain.MockedCurrentBlock + 20}

	_, err := f.service.GetListUnclaimed(context.Background(), req)
	assert.Error(t, err)
}

func TestGetListInProgress(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) { return nil, nil }
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{Expiration: big.NewInt(100)}, true, nil
	}

	req := &GetPaymentsListRequest{MpeAddress: f.svcMD.MpeAddress, CurrentBlock: blockchain.MockedCurrentBlock}
	req.Signature = f.signListRequest("__list_in_progress", req.CurrentBlock)

	reply, err := f.service.GetListInProgress(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
}

func TestStartClaim(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) { return nil, nil }
	f.channelMock.paymentChannelFn = func(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{ChannelID: key.ID, Nonce: big.NewInt(3), AuthorizedAmount: big.NewInt(10)}, true, nil
	}
	f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
		return &mockClaim{payment: &Payment{ChannelNonce: big.NewInt(3), Amount: big.NewInt(10), Signature: []byte{1}}}, nil
	}

	channelID := big.NewInt(42)
	req := &StartClaimRequest{MpeAddress: f.svcMD.MpeAddress, ChannelId: bigIntToBytes(channelID)}
	message := bytes.Join([][]byte{
		[]byte("__start_claim"),
		f.svcMD.GetMpeAddress().Bytes(),
		bigIntToBytes(channelID),
		bigIntToBytes(big.NewInt(3)),
	}, nil)
	req.Signature = getSignature(message, f.signerKey)

	reply, err := f.service.StartClaim(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), bytesToBigInt(reply.ChannelId).Int64())
}

func TestStartClaimForMultipleChannels(t *testing.T) {
	f := newControlServiceFixture(t)
	f.channelMock.listClaimsFn = func() ([]Claim, error) { return nil, nil }
	f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{AuthorizedAmount: big.NewInt(10), Nonce: big.NewInt(0)}, true, nil
	}
	f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
		return &mockClaim{payment: &Payment{ChannelNonce: big.NewInt(0), Amount: big.NewInt(10)}}, nil
	}

	req := &StartMultipleClaimRequest{
		MpeAddress:   f.svcMD.MpeAddress,
		ChannelIds:   []uint64{2, 1},
		CurrentBlock: blockchain.MockedCurrentBlock,
	}
	message := bytes.Join([][]byte{
		[]byte("__StartClaimForMultipleChannels_"),
		f.svcMD.GetMpeAddress().Bytes(),
		bigIntToBytes(big.NewInt(1)),
		bigIntToBytes(big.NewInt(2)),
		math.U256Bytes(big.NewInt(int64(req.CurrentBlock))),
	}, nil)
	req.Signature = getSignature(message, f.signerKey)

	reply, err := f.service.StartClaimForMultipleChannels(context.Background(), req)
	assert.NoError(t, err)
	require.Len(t, reply.Payments, 2)
}
