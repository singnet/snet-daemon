package escrow

type lockerMock struct {
}

func (mock *lockerMock) Lock(name string) (lock Lock, ok bool, err error) {
	return &lockMock{}, true, nil
}

type lockMock struct {
}

func (mock *lockMock) Unlock() (err error) {
	return nil
}
