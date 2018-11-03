package escrow

// AtomicStorage is an interface to key-value storage with atomic operations.
type AtomicStorage interface {
	// Get returns value by key. ok value indicates whether passed key is
	// present in the storage. err indicates storage error.
	Get(key string) (value string, ok bool, err error)
	// Put uncoditionally writes value by key in storage, err is not nil in
	// case of storage error.
	Put(key string, value string) (err error)
	// PutIfAbsent writes value if and only if key is absent in storage. ok is
	// true if key was absent and false otherwise. err indicates storage error.
	PutIfAbsent(key string, value string) (ok bool, err error)
	// CompareAndSwap atomically replaces prevValue by newValue. If ok flag is
	// true and err is nil then operation was successful. If err is nil and ok
	// is false then operation failed because prevValue is not equal to current
	// value. err indicates storage error.
	CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error)
}

type PrefixedAtomicStorage struct {
	delegate  AtomicStorage
	keyPrefix string
}

func (storage *PrefixedAtomicStorage) Get(key string) (value string, ok bool, err error) {
	return storage.delegate.Get(storage.keyPrefix + "-" + key)
}

func (storage *PrefixedAtomicStorage) Put(key string, value string) (err error) {
	return storage.delegate.Put(storage.keyPrefix+"-"+key, value)
}

func (storage *PrefixedAtomicStorage) PutIfAbsent(key string, value string) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(storage.keyPrefix+"-"+key, value)
}

func (storage *PrefixedAtomicStorage) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(storage.keyPrefix+"-"+key, prevValue, newValue)
}
