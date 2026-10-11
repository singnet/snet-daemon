package freecall

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/require"
)

func TestParseTokenBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name, token, signature, block, wantErr string
	}{
		{name: "missing separator", token: "token", wantErr: "no '_' found"},
		{name: "empty expiration", token: "token_", wantErr: "invalid block number"},
		{name: "invalid expiration", token: "token_x", wantErr: "invalid block number"},
		{name: "separator in signature", token: "a_b_100", signature: "a_b", block: "100"},
		{name: "large expiration", token: "sig_18446744073709551616", signature: "sig", block: "18446744073709551616"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sig, block, err := ParseToken([]byte(tt.token))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, sig)
				require.Nil(t, block)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.signature, string(sig))
			require.Equal(t, tt.block, block.String())
		})
	}
}

func TestNewTokenLifetime(t *testing.T) {
	for _, tt := range []struct {
		name     string
		lifetime *uint64
		want     int64
	}{
		{name: "default", want: 100 + TokenLifetime},
		{name: "zero", lifetime: new(uint64), want: 100},
		{name: "custom", lifetime: uint64Pointer(20), want: 120},
		{name: "maximum", lifetime: uint64Pointer(TokenLifetime), want: 100 + TokenLifetime},
		{name: "over maximum", lifetime: uint64Pointer(TokenLifetime + 1), want: 100 + TokenLifetime},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			user := crypto.PubkeyToAddress(f.userKey.PublicKey)
			token, expiration := f.validator.NewToken(user.Hex(), nil, tt.lifetime)
			require.Equal(t, big.NewInt(tt.want), expiration)
			sig, parsedExpiration, err := ParseToken(token)
			require.NoError(t, err)
			require.Equal(t, expiration, parsedExpiration)
			signer, err := utils.GetSignerAddressFromMessage(BuildTokenMessage(&user, expiration, nil), sig)
			require.NoError(t, err)
			require.Equal(t, f.validator.freeCallSignerAddress, *signer)
		})
	}
}

func uint64Pointer(value uint64) *uint64 { return &value }

func TestValidatorBlockchainErrors(t *testing.T) {
	f := newAuthFixture(t)
	wantErr := errors.New("blockchain unavailable")
	f.validator.currentBlock = func() (*big.Int, error) { return nil, wantErr }
	token, expiration := f.validator.NewToken(common.Address{}.Hex(), nil, nil)
	require.Nil(t, token)
	require.Nil(t, expiration)
	require.ErrorIs(t, f.validator.CheckIfBlockExpired(big.NewInt(100)), wantErr)
	require.ErrorIs(t, f.validator.compareWithLatestBlockNumber(big.NewInt(100)), wantErr)
}

func TestCompareSignerAddresses(t *testing.T) {
	f := newAuthFixture(t)
	trusted := crypto.PubkeyToAddress(f.userKey.PublicKey)
	f.validator.trustedFreeCallSignerAddresses = []common.Address{trusted}
	require.NoError(t, f.validator.CompareSignerAddrs(trusted))
	require.NoError(t, f.validator.CompareSignerAddrs(f.validator.freeCallSignerAddress))
	err := f.validator.CompareSignerAddrs(common.Address{})
	var paymentErr *errs.PaymentError
	require.ErrorAs(t, err, &paymentErr)
	require.Equal(t, errs.Unauthenticated, paymentErr.Code)
}

func TestValidateRejectsInvalidAuthorization(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*authFixture, *Payment)
		wantErr string
	}{
		{"request signature", func(_ *authFixture, p *Payment) { p.Signature = []byte{1} }, "sign is not valid"},
		{"request signer", func(f *authFixture, p *Payment) { signPaymentRequest(p, f.signerKey) }, "unauthorized signer"},
		{"token signature", func(f *authFixture, p *Payment) {
			p.AuthTokenParsed = []byte{1}
			p.AuthToken = append([]byte{1}, []byte("_"+p.AuthTokenExpiryBlockNumber.String())...)
			signPaymentRequest(p, f.userKey)
		}, "sign is not valid"},
		{"token signer", func(f *authFixture, p *Payment) {
			addr := common.HexToAddress(p.Address)
			p.AuthTokenParsed = getSignature(BuildTokenMessage(&addr, p.AuthTokenExpiryBlockNumber, &p.UserID), f.userKey)
			p.AuthToken = append(append([]byte(nil), p.AuthTokenParsed...), []byte("_"+p.AuthTokenExpiryBlockNumber.String())...)
			signPaymentRequest(p, f.userKey)
		}, "token sign is not valid"},
		{"curation", func(_ *authFixture, _ *Payment) { config.Vip().Set(config.AllowedUserFlag, true) }, "not Authorized"},
		{"expired token", func(f *authFixture, _ *Payment) {
			f.validator.currentBlock = func() (*big.Int, error) { return big.NewInt(100 + TokenLifetime + 1), nil }
		}, "Token passed has expired"},
		{"stale request", func(f *authFixture, p *Payment) {
			p.CurrentBlockNumber = big.NewInt(94)
			signPaymentRequest(p, f.userKey)
		}, "signature passed has expired"},
		{"future request", func(f *authFixture, p *Payment) {
			p.CurrentBlockNumber = big.NewInt(106)
			signPaymentRequest(p, f.userKey)
		}, "signature passed has expired"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			payment := f.payment(t)
			tt.change(f, payment)
			require.ErrorContains(t, f.validator.Validate(payment), tt.wantErr)
		})
	}
}

func TestBlockValidityBoundaries(t *testing.T) {
	f := newAuthFixture(t)
	for _, block := range []int64{95, 100, 105} {
		require.NoError(t, f.validator.compareWithLatestBlockNumber(big.NewInt(block)))
	}
	require.NoError(t, f.validator.CheckIfBlockExpired(big.NewInt(100)))
	require.NoError(t, f.validator.CheckIfBlockExpired(big.NewInt(101)))
	require.Error(t, f.validator.CheckIfBlockExpired(big.NewInt(99)))
}
