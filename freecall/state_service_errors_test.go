package freecall

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

type balanceReaderFunc func(*bind.CallOpts, common.Address) (*big.Int, error)

func (f balanceReaderFunc) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	return f(opts, account)
}

type stateUserService struct {
	UserService
	keyErr, userErr error
	missing         bool
}

func (s *stateUserService) UserKey(payment *Payment) (*UserKey, error) {
	if s.keyErr != nil {
		return nil, s.keyErr
	}
	return s.UserService.UserKey(payment)
}

func (s *stateUserService) User(key *UserKey) (*UserData, bool, error) {
	if s.userErr != nil || s.missing {
		return nil, false, s.userErr
	}
	return s.UserService.User(key)
}

func (f *authFixture) stateService() (*StateService, *UserStorage) {
	mem := storage.NewMemStorage()
	users := NewUserStorage(mem)
	service := NewUserService(users, storage.NewLocker(mem), f.groupID, f.metadata)
	return NewStateService(f.org, f.metadata, service, f.validator, MockedERC20{}, big.NewInt(1)), users
}

func TestCheckBalanceForFreeCall(t *testing.T) {
	threshold := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	for _, tt := range []struct {
		name    string
		balance *big.Int
		err     error
		allowed bool
	}{
		{"unavailable", nil, errors.New("RPC unavailable"), false},
		{"below threshold", new(big.Int).Sub(threshold, big.NewInt(1)), nil, false},
		{"exact threshold", new(big.Int).Set(threshold), nil, true},
		{"above threshold", new(big.Int).Add(threshold, big.NewInt(1)), nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			service, _ := f.stateService()
			addr := crypto.PubkeyToAddress(f.userKey.PublicKey)
			ctx := t.Context()
			service.tokenInstance = balanceReaderFunc(func(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
				require.Equal(t, ctx, opts.Context)
				require.Equal(t, addr, account)
				return tt.balance, tt.err
			})
			err := service.CheckBalanceForFreeCall(ctx, addr)
			if tt.allowed {
				require.NoError(t, err)
			} else {
				var grpcErr *handler.GrpcError
				require.ErrorAs(t, err, &grpcErr)
				require.Equal(t, codes.PermissionDenied, grpcErr.Status.Code())
			}
			require.Equal(t, big.NewInt(1), service.minBalanceForFreeCall)
		})
	}
}

func TestGetFreeCallTokenRejections(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*authFixture, *StateService, *GetFreeCallTokenRequest)
		wantErr string
	}{
		{"invalid signature", func(_ *authFixture, _ *StateService, r *GetFreeCallTokenRequest) { r.Signature = []byte{1} }, "signature"},
		{"wrong signer", func(f *authFixture, _ *StateService, r *GetFreeCallTokenRequest) {
			signTokenRequest(r, f.org.GetGroupIdString(), f.signerKey)
		}, "invalid signer"},
		{"curation", func(_ *authFixture, _ *StateService, _ *GetFreeCallTokenRequest) {
			config.Vip().Set(config.AllowedUserFlag, true)
		}, "not Authorized"},
		{"expired request", func(f *authFixture, _ *StateService, r *GetFreeCallTokenRequest) {
			r.CurrentBlock = 94
			signTokenRequest(r, f.org.GetGroupIdString(), f.userKey)
		}, "signature passed has expired"},
		{"untrusted user ID", func(f *authFixture, _ *StateService, r *GetFreeCallTokenRequest) {
			r.UserId = new(string)
			*r.UserId = "untrusted-user"
			signTokenRequest(r, f.org.GetGroupIdString(), f.userKey)
		}, "user_id is not allowed"},
		{"insufficient balance", func(_ *authFixture, s *StateService, _ *GetFreeCallTokenRequest) {
			s.minBalanceForFreeCall = big.NewInt(2)
		}, "at least 2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			service, _ := f.stateService()
			request := &GetFreeCallTokenRequest{Address: crypto.PubkeyToAddress(f.userKey.PublicKey).Hex(), CurrentBlock: 100}
			signTokenRequest(request, f.org.GetGroupIdString(), f.userKey)
			tt.change(f, service, request)
			reply, err := service.GetFreeCallToken(t.Context(), request)
			require.Nil(t, reply)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTrustedSignerCanRequestTokenForUser(t *testing.T) {
	f := newAuthFixture(t)
	service, _ := f.stateService()
	address := crypto.PubkeyToAddress(f.userKey.PublicKey)
	f.validator.trustedFreeCallSignerAddresses = []common.Address{address}
	service.tokenInstance = balanceReaderFunc(func(*bind.CallOpts, common.Address) (*big.Int, error) {
		t.Fatal("trusted signer must not require a balance lookup")
		return nil, nil
	})
	userID := "marketplace-user"
	request := &GetFreeCallTokenRequest{Address: address.Hex(), UserId: &userID, CurrentBlock: 100, TokenLifetimeInBlocks: uint64Pointer(20)}
	signTokenRequest(request, f.org.GetGroupIdString(), f.userKey)
	reply, err := service.GetFreeCallToken(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, uint64(120), reply.TokenExpirationBlock)
	payment := f.payment(t)
	payment.UserID = userID
	payment.AuthToken = reply.Token
	payment.AuthTokenParsed, payment.AuthTokenExpiryBlockNumber, err = ParseToken(reply.Token)
	require.NoError(t, err)
	signPaymentRequest(payment, f.userKey)
	require.NoError(t, f.validator.Validate(payment))
}

func TestFreeCallsAvailableLimits(t *testing.T) {
	for _, tt := range []struct {
		name        string
		limit, used int
		want        uint64
	}{
		{"metadata", 0, 3, 7}, {"metadata exhausted", 0, 10, 0}, {"metadata exceeded", 0, 11, 0},
		{"override", 5, 3, 2}, {"override exhausted", 5, 5, 0}, {"override exceeded", 5, 6, 0},
		{"unlimited", -1, 100, 99999999},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			service, users := f.stateService()
			payment := f.payment(t)
			config.Vip().Set(config.FreeCallsPerAddress, map[string]any{payment.Address: tt.limit})
			key, err := service.freeCallService.UserKey(payment)
			require.NoError(t, err)
			require.NoError(t, users.Put(key, &UserData{FreeCallsMade: tt.used}))
			reply, err := service.GetFreeCallsAvailable(t.Context(), stateRequest(payment))
			require.NoError(t, err)
			require.Equal(t, tt.want, reply.FreeCallsAvailable)
			stored, ok, err := users.Get(key)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, tt.used, stored.FreeCallsMade, "query must not consume a call")
		})
	}
}

func stateRequest(p *Payment) *FreeCallStateRequest {
	return &FreeCallStateRequest{Address: p.Address, UserId: &p.UserID, CurrentBlock: p.CurrentBlockNumber.Uint64(), Signature: p.Signature, FreeCallToken: p.AuthToken}
}

func TestFreeCallsAvailableStorageErrors(t *testing.T) {
	for _, tt := range []struct {
		name            string
		keyErr, userErr error
		missing         bool
		wantErr         string
	}{
		{name: "key", keyErr: errors.New("group unavailable"), wantErr: "group unavailable"},
		{name: "user", userErr: errors.New("storage unavailable"), wantErr: "storage unavailable"},
		{name: "missing user", missing: true, wantErr: "error in retrieving free call details"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			service, _ := f.stateService()
			service.freeCallService = &stateUserService{UserService: service.freeCallService, keyErr: tt.keyErr, userErr: tt.userErr, missing: tt.missing}
			_, err := service.GetFreeCallsAvailable(t.Context(), stateRequest(f.payment(t)))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestFreeCallsAvailableMalformedToken(t *testing.T) {
	f := newAuthFixture(t)
	service, _ := f.stateService()
	reply, err := service.GetFreeCallsAvailable(t.Context(), &FreeCallStateRequest{FreeCallToken: []byte("invalid")})
	require.Nil(t, reply)
	require.EqualError(t, err, "no '_' found")
}
