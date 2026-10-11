package escrow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func isolateConfig(t *testing.T) {
	t.Helper()
	previous := config.Vip()
	config.SetVip(viper.New())
	t.Cleanup(func() { config.SetVip(previous) })
}

type recordingLock struct {
	err     error
	unlocks int
}

func (l *recordingLock) Unlock() error {
	l.unlocks++
	return l.err
}

type lockerFunc func(string) (storage.Lock, bool, error)

func (f lockerFunc) Lock(name string) (storage.Lock, bool, error) { return f(name) }

type transactionFixture struct {
	service            *lockingPaymentChannelService
	channel            *PaymentChannelData
	payment            *Payment
	channels, payments *failingStorage
	lock               *recordingLock
}

func newTransactionFixture(t *testing.T) *transactionFixture {
	t.Helper()
	isolateConfig(t)
	f := newTokenServiceFixture(t)
	channels := &failingStorage{MemoryStorage: storage.NewMemStorage()}
	payments := &failingStorage{MemoryStorage: storage.NewMemStorage()}
	channel := f.channel()
	channel.Signature = []byte{1, 2, 3}
	lock := &recordingLock{}
	service := &lockingPaymentChannelService{
		storage:          NewPaymentChannelStorage(channels),
		paymentStorage:   NewPaymentStorage(payments),
		blockchainReader: NewBlockchainChannelReaderMock(),
		locker: lockerFunc(func(name string) (storage.Lock, bool, error) {
			require.Equal(t, (&PaymentChannelKey{ID: channel.ChannelID}).String(), name)
			return lock, true, nil
		}),
		validator:      f.service.validator,
		replicaGroupID: func() [32]byte { return channel.GroupID },
	}
	payment := &Payment{ChannelID: channel.ChannelID, ChannelNonce: channel.Nonce, Amount: big.NewInt(11)}
	SignTestPayment(payment, f.senderKey)
	require.NoError(t, service.storage.Put(&PaymentChannelKey{ID: channel.ChannelID}, channel))
	return &transactionFixture{service: service, channel: channel, payment: payment, channels: channels, payments: payments, lock: lock}
}

func TestStartPaymentTransactionErrorsReleaseLock(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*transactionFixture)
		code    PaymentErrorCode
		unlocks int
	}{
		{"lock error", func(f *transactionFixture) {
			f.service.locker = lockerFunc(func(string) (storage.Lock, bool, error) { return nil, false, errors.New("lock failed") })
		}, Internal, 0},
		{"lock busy", func(f *transactionFixture) {
			f.service.locker = lockerFunc(func(string) (storage.Lock, bool, error) { return nil, false, nil })
		}, FailedPrecondition, 0},
		{"storage error", func(f *transactionFixture) { f.channels.getErr = errors.New("read failed") }, Internal, 1},
		{"missing channel", func(f *transactionFixture) { f.channels.Clear() }, Unauthenticated, 1},
		{"invalid payment", func(f *transactionFixture) { f.payment.ChannelNonce = big.NewInt(99) }, IncorrectNonce, 1},
		{"unlock error", func(f *transactionFixture) {
			f.payment.ChannelNonce = big.NewInt(99)
			f.lock.err = errors.New("unlock failed")
		}, IncorrectNonce, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newTransactionFixture(t)
			tt.change(f)
			transaction, err := f.service.StartPaymentTransaction(f.payment)
			require.Nil(t, transaction)
			var paymentErr *PaymentError
			require.ErrorAs(t, err, &paymentErr)
			require.Equal(t, tt.code, paymentErr.Code)
			require.Equal(t, tt.unlocks, f.lock.unlocks)
			f.channels.getErr = nil
			stored, ok, err := f.service.storage.Get(&PaymentChannelKey{ID: f.channel.ChannelID})
			require.NoError(t, err)
			if ok {
				require.Equal(t, f.channel, stored, "rejected payment must not update the channel")
			}
		})
	}
}

func TestPaymentTransactionFinalization(t *testing.T) {
	for _, tt := range []struct {
		name                            string
		commit, writeError, unlockError bool
	}{
		{"commit", true, false, false},
		{"commit storage failure", true, true, false},
		{"commit unlock failure", true, false, true},
		{"rollback", false, false, false},
		{"rollback unlock failure", false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newTransactionFixture(t)
			transaction, err := f.service.StartPaymentTransaction(f.payment)
			require.NoError(t, err)
			payment := transaction.(*paymentTransaction)
			require.Equal(t, f.channel.Sender, payment.GetSender())
			require.Contains(t, payment.String(), "{payment:")
			require.Contains(t, payment.String(), f.channel.String())
			if tt.writeError {
				f.channels.putErr = errors.New("write failed")
			}
			if tt.unlockError {
				f.lock.err = errors.New("unlock failed")
			}
			if tt.commit {
				err = transaction.Commit()
			} else {
				err = transaction.Rollback()
			}
			if tt.writeError {
				var paymentErr *PaymentError
				require.ErrorAs(t, err, &paymentErr)
				require.Equal(t, Internal, paymentErr.Code)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, f.lock.unlocks)
			stored, ok, err := f.service.storage.Get(&PaymentChannelKey{ID: f.channel.ChannelID})
			require.NoError(t, err)
			require.True(t, ok)
			want := *f.channel
			if tt.commit && !tt.writeError {
				want.AuthorizedAmount = f.payment.Amount
				want.Signature = f.payment.Signature
			}
			require.Equal(t, &want, stored)
		})
	}
}

func TestStartClaimFailures(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*transactionFixture)
		wantErr string
		unlocks int
	}{
		{"lock error", func(f *transactionFixture) {
			f.service.locker = lockerFunc(func(string) (storage.Lock, bool, error) { return nil, false, errors.New("lock failed") })
		}, "cannot get mutex", 0},
		{"lock busy", func(f *transactionFixture) {
			f.service.locker = lockerFunc(func(string) (storage.Lock, bool, error) { return nil, false, nil })
		}, "another transaction", 0},
		{"storage read", func(f *transactionFixture) { f.channels.getErr = errors.New("read failed") }, "read failed", 1},
		{"missing channel", func(f *transactionFixture) { f.channels.Clear() }, "Channel is not found", 1},
		{"channel write", func(f *transactionFixture) { f.channels.putErr = errors.New("channel write failed") }, "Channel storage error", 1},
		{"payment write", func(f *transactionFixture) { f.payments.putErr = errors.New("payment write failed") }, "payment write failed", 1},
		{"unlock failure", func(f *transactionFixture) { f.lock.err = errors.New("unlock failed") }, "", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newTransactionFixture(t)
			tt.change(f)
			claim, err := f.service.StartClaim(&PaymentChannelKey{ID: f.channel.ChannelID}, IncrementChannelNonce)
			if tt.wantErr != "" {
				require.Nil(t, claim)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, f.channel.AuthorizedAmount, claim.Payment().Amount)
			}
			require.Equal(t, tt.unlocks, f.lock.unlocks)
		})
	}
}

func TestClaimFinishRemovesOnlyClaimedPayment(t *testing.T) {
	f := newTransactionFixture(t)
	other := &Payment{ChannelID: big.NewInt(2), ChannelNonce: big.NewInt(0), Amount: big.NewInt(7)}
	require.NoError(t, f.service.paymentStorage.Put(other))
	claim, err := f.service.StartClaim(&PaymentChannelKey{ID: f.channel.ChannelID}, IncrementChannelNonce)
	require.NoError(t, err)
	require.NoError(t, claim.Finish())
	claims, err := f.service.ListClaims()
	require.NoError(t, err)
	require.Len(t, claims, 1)
	require.Equal(t, other, claims[0].Payment())
	_, ok, err := f.service.paymentStorage.Get(claim.Payment().ID())
	require.NoError(t, err)
	require.False(t, ok)
}

func TestChannelAndPaymentStorageFailures(t *testing.T) {
	f := newTransactionFixture(t)
	wantErr := errors.New("storage unavailable")
	f.channels.getErr = wantErr
	channels, err := f.service.ListChannels()
	require.Nil(t, channels)
	require.ErrorIs(t, err, wantErr)
	f.payments.getErr = wantErr
	claims, err := f.service.ListClaims()
	require.Nil(t, claims)
	require.ErrorIs(t, err, wantErr)
	payment, ok, err := f.service.paymentStorage.Get(f.payment.ID())
	require.Nil(t, payment)
	require.False(t, ok)
	require.ErrorIs(t, err, wantErr)
}

func TestChannelStorageConditionalUpdates(t *testing.T) {
	f := newTransactionFixture(t)
	key := &PaymentChannelKey{ID: f.channel.ChannelID}
	revised := *f.channel
	revised.AuthorizedAmount = big.NewInt(15)
	ok, err := f.service.storage.PutIfAbsent(key, &revised)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = f.service.storage.CompareAndSwap(key, &revised, f.channel)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = f.service.storage.CompareAndSwap(key, f.channel, &revised)
	require.NoError(t, err)
	require.True(t, ok)
	stored, ok, err := f.service.storage.Get(key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, &revised, stored)
	f.channels.Clear()
	ok, err = f.service.storage.PutIfAbsent(key, f.channel)
	require.NoError(t, err)
	require.True(t, ok)
}
