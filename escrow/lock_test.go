package escrow

import "github.com/singnet/snet-daemon/v6/storage"

type lockerMock struct {
}

func (mock *lockerMock) Lock(name string) (lock storage.Lock, ok bool, err error) {
	return &lockMock{}, true, nil
}

type lockMock struct {
}

func (mock *lockMock) Unlock() (err error) {
	return nil
}
