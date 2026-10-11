package storage

import (
	"go.uber.org/zap"
)

// Lock is an acquired lock.
type Lock interface {
	// Unlock frees lock
	Unlock() (err error)
}

// Locker is an interface to acquire lock
type Locker interface {
	// Lock acquires and returns lock. ok is false if lock cannot be acquired.
	Lock(name string) (lock Lock, ok bool, err error)
}

// NewLocker returns a locker backed by atomic storage.
func NewLocker(storage AtomicStorage) Locker {
	return &atomicLocker{
		storage: storage,
	}
}

type atomicLocker struct {
	storage AtomicStorage
}

const (
	locked   = "locked"
	unlocked = "unlocked"
)

func (locker *atomicLocker) Lock(name string) (lock Lock, ok bool, err error) {
	value, ok, err := locker.storage.Get(name)
	if err != nil {
		return
	}
	if ok {
		if value == locked {
			return nil, false, nil
		}
		ok, err = locker.storage.CompareAndSwap(name, unlocked, locked)
	} else {
		ok, err = locker.storage.PutIfAbsent(name, locked)
	}

	if err != nil || !ok {
		return
	}

	return &lockType{
		name:   name,
		locker: locker,
	}, true, nil
}

type lockType struct {
	name   string
	locker *atomicLocker
}

func (lock *lockType) Unlock() (err error) {
	ok, err := lock.locker.storage.CompareAndSwap(lock.name, locked, unlocked)
	if err != nil {
		return
	}
	if !ok {
		zap.L().Error("lock is unlocked already", zap.String("lock.name", lock.name))
	}
	return
}
