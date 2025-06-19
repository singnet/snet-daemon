package escrow

import (
	"github.com/singnet/snet-daemon/v6/storage"
	"reflect"
)

// PaymentStorage is a storage for PaymentChannelData by
// PaymentChannelKey based on TypedAtomicStorage implementation
type PaymentStorage struct {
	delegate storage.TypedAtomicStorage
}

// NewPaymentStorage returns new instance of PaymentStorage
// implementation
func NewPaymentStorage(atomicStorage storage.AtomicStorage) *PaymentStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/payment/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeKey, reflect.TypeOf(""), serialize, deserialize,
		reflect.TypeOf(Payment{}),
	)
	return &PaymentStorage{delegate: storage}
}

func (storage *PaymentStorage) GetAll() (states []*Payment, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*Payment), nil
}

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

func (storage *PaymentStorage) Put(payment *Payment) (err error) {
	return storage.delegate.Put(payment.ID(), payment)
}

func (storage *PaymentStorage) Delete(payment *Payment) (err error) {
	return storage.delegate.Delete(payment.ID())
}
