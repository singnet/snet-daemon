package escrow

import (
	log "github.com/sirupsen/logrus"
)

type Lock interface {
	Unlock() (err error)
}

type Locker interface {
	Lock(name string) (lock Lock, ok bool, err error)
}

func NewEtcdLocker(storage AtomicStorage) Locker {
	return &etcdLocker{
		storage: &PrefixedAtomicStorage{
			delegate:  storage,
			keyPrefix: "lock",
		},
	}
}

type etcdLocker struct {
	storage AtomicStorage
}

const (
	locked   = "locked"
	unlocked = "unlocked"
)

func (locker *etcdLocker) Lock(name string) (lock Lock, ok bool, err error) {
	_, ok, err = locker.storage.Get(name)
	if err != nil {
		return
	}
	if ok {
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
