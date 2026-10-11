package freecall

import (
	"errors"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestPaymentHandlerRejectsInvalidMetadata(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(metadata.MD)
		code   codes.Code
	}{
		{"missing address", func(md metadata.MD) { md.Delete(handler.FreeCallUserAddressHeader) }, codes.InvalidArgument},
		{"missing block", func(md metadata.MD) { md.Delete(handler.CurrentBlockNumberHeader) }, codes.InvalidArgument},
		{"invalid block", func(md metadata.MD) { md.Set(handler.CurrentBlockNumberHeader, "NaN") }, codes.InvalidArgument},
		{"missing signature", func(md metadata.MD) { md.Delete(handler.PaymentChannelSignatureHeader) }, codes.InvalidArgument},
		{"missing token", func(md metadata.MD) { md.Delete(handler.FreeCallAuthTokenHeader) }, codes.InvalidArgument},
		{"malformed token", func(md metadata.MD) { md.Set(handler.FreeCallAuthTokenHeader, "invalid") }, codes.InvalidArgument},
		{"invalid signature", func(md metadata.MD) { md.Set(handler.PaymentChannelSignatureHeader, "invalid") }, codes.Unauthenticated},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			payment := f.payment(t)
			md := metadata.Pairs(
				handler.FreeCallUserAddressHeader, payment.Address,
				handler.FreeCallUserIdHeader, payment.UserID,
				handler.CurrentBlockNumberHeader, payment.CurrentBlockNumber.String(),
				handler.PaymentChannelSignatureHeader, string(payment.Signature),
				handler.FreeCallAuthTokenHeader, string(payment.AuthToken),
			)
			tt.change(md)
			// No service is provided: authentication must fail before accessing storage.
			h := NewPaymentHandler(nil, blockchain.NewMockProcessor(true), f.org, f.metadata)
			result, err := h.Payment(&handler.GrpcStreamContext{MD: md})
			require.Nil(t, result)
			require.NotNil(t, err)
			require.Equal(t, tt.code, err.Status.Code())
		})
	}
}

type recordingLock struct {
	err     error
	unlocks int
}

func (l *recordingLock) Unlock() error {
	l.unlocks++
	return l.err
}

func TestRejectedTransactionAttemptsUnlock(t *testing.T) {
	f := newAuthFixture(t)
	users := NewUserStorage(storage.NewMemStorage())
	lock := &recordingLock{err: errors.New("unlock unavailable")}
	service := NewUserService(users, mockLocker{lock: lock, ok: true}, f.groupID, f.metadata)
	payment := f.payment(t)
	key, err := service.UserKey(payment)
	require.NoError(t, err)
	require.NoError(t, users.Put(key, &UserData{FreeCallsMade: 10}))
	transaction, err := service.StartTransaction(payment)
	require.Nil(t, transaction)
	require.ErrorContains(t, err, "free call limit has been exceeded")
	require.Equal(t, 1, lock.unlocks)
	stored, ok, err := users.Get(key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 10, stored.FreeCallsMade)
}

func TestListUsersStorageFailure(t *testing.T) {
	f := newAuthFixture(t)
	wantErr := errors.New("storage unavailable")
	users := NewUserStorage(&failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: wantErr})
	service := NewUserService(users, nil, f.groupID, f.metadata)
	result, err := service.ListUsers()
	require.Nil(t, result)
	require.ErrorIs(t, err, wantErr)
}
