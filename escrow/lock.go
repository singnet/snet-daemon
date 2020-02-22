package escrow

import (
	log "github.com/sirupsen/logrus"
)

// Lock is an aquired lock.
type Lock interface {
	// Unlock frees lock
	Unlock() (err error)
}

// Locker is an interface to aquire lock
type Locker interface {
	// Lock aquires and returns lock. ok is false if lock cannot be aquired.
	Lock(name string) (lock Lock, ok bool, err error)
}

// NewEtcdLocker returns new lock which is based on etcd storage.
func NewEtcdLocker(storage AtomicStorage) Locker {
	return &etcdLocker{
		storage: NewLockerStorage(storage),
	}
}

// returns new prefixed storage
func NewLockerStorage(storage AtomicStorage) *PrefixedAtomicStorage {
	return NewPrefixedAtomicStorage(storage, "/payment-channel/lock")
}

type etcdLocker struct {
	storage AtomicStorage
}

const (
	locked   = "locked"
	unlocked = "unlocked"
)

func (locker *etcdLocker) Lock(name string) (lock Lock, ok bool, err error) {
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
	locker *etcdLocker
}

func (lock *lockType) Unlock() (err error) {
	ok, err := lock.locker.storage.CompareAndSwap(lock.name, locked, unlocked)
	if err != nil {
		return
	}
	if !ok {
		log.WithField("lock.name", lock.name).Error("lock is unlocked already")
	}
	return
}
