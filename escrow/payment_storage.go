package escrow

import (
	"reflect"

	"github.com/singnet/snet-daemon/v6/storage"
)

// PaymentStorage stores pending claim Payments by PaymentID (channel ID/nonce).
// It is separate from PaymentChannelStorage, which stores current channel state
// by channel ID alone.
type PaymentStorage struct {
	delegate storage.TypedAtomicStorage
}

// NewPaymentStorage creates typed payment storage under the /payment/storage prefix.
func NewPaymentStorage(atomicStorage storage.AtomicStorage) *PaymentStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/payment/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeKey, reflect.TypeFor[string](), storage.EncodeGob, storage.DecodeGob,
		reflect.TypeFor[Payment](),
	)
	return &PaymentStorage{delegate: storage}
}

// GetAll returns all pending claim payments.
func (storage *PaymentStorage) GetAll() (states []*Payment, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*Payment), nil
}

// Get returns the pending payment identified by PaymentID, with ok false if absent.
func (storage *PaymentStorage) Get(paymentID string) (payment *Payment, ok bool, err error) {
	value, ok, err := storage.delegate.Get(paymentID)
	if err != nil {
		return
	}
	if !ok {
		return
	}
	return value.(*Payment), true, nil
}

// Put stores a pending payment under its channel ID and nonce.
func (storage *PaymentStorage) Put(payment *Payment) (err error) {
	return storage.delegate.Put(payment.ID(), payment)
}

// Delete removes the pending payment for the supplied channel ID and nonce.
func (storage *PaymentStorage) Delete(payment *Payment) (err error) {
	return storage.delegate.Delete(payment.ID())
}
