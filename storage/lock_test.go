package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockerLifecycleAndIndependentNames(t *testing.T) {
	store := NewMemStorage()
	locker := NewLocker(store)
	first, ok, err := locker.Lock("first")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, first)
	busy, ok, err := NewLocker(store).Lock("first")
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, busy)
	second, ok, err := locker.Lock("second")
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, second.Unlock())
	require.NoError(t, first.Unlock())
	// An immediate repeated unlock leaves the key available for the next acquisition.
	require.NoError(t, first.Unlock())
	again, ok, err := NewLocker(store).Lock("first")
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, again.Unlock())
}

func TestLockerConcurrentAcquisitionHasOneWinner(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new key"
		if existing {
			name = "released key"
		}
		t.Run(name, func(t *testing.T) {
			store := NewMemStorage()
			if existing {
				require.NoError(t, store.Put("shared", unlocked))
			}
			type result struct {
				lock Lock
				ok   bool
				err  error
			}
			const contenders = 32
			start := make(chan struct{})
			results := make(chan result, contenders)
			for range contenders {
				go func() {
					<-start
					lock, ok, err := NewLocker(store).Lock("shared")
					results <- result{lock, ok, err}
				}()
			}
			close(start)
			var winner Lock
			wins := 0
			for range contenders {
				got := <-results
				require.NoError(t, got.err)
				if got.ok {
					require.NotNil(t, got.lock)
					wins++
					winner = got.lock
				} else {
					require.Nil(t, got.lock)
				}
			}
			require.Equal(t, 1, wins)
			require.NoError(t, winner.Unlock())
			next, ok, err := NewLocker(store).Lock("shared")
			require.NoError(t, err)
			require.True(t, ok)
			require.NoError(t, next.Unlock())
		})
	}
}

type lockerStorageHooks struct {
	AtomicStorage
	get            func(string) (string, bool, error)
	putIfAbsent    func(string, string) (bool, error)
	compareAndSwap func(string, string, string) (bool, error)
}

func (s *lockerStorageHooks) Get(key string) (string, bool, error) {
	if s.get != nil {
		return s.get(key)
	}
	return s.AtomicStorage.Get(key)
}
func (s *lockerStorageHooks) PutIfAbsent(key, value string) (bool, error) {
	if s.putIfAbsent != nil {
		return s.putIfAbsent(key, value)
	}
	return s.AtomicStorage.PutIfAbsent(key, value)
}
func (s *lockerStorageHooks) CompareAndSwap(key, previous, next string) (bool, error) {
	if s.compareAndSwap != nil {
		return s.compareAndSwap(key, previous, next)
	}
	return s.AtomicStorage.CompareAndSwap(key, previous, next)
}

func TestLockerStorageErrorsAndLostRaces(t *testing.T) {
	failure := errors.New("lock storage unavailable")
	for _, scenario := range []string{"read error", "insert error", "insert race", "swap error", "swap race"} {
		t.Run(scenario, func(t *testing.T) {
			store := &lockerStorageHooks{AtomicStorage: NewMemStorage()}
			var wantErr error
			if scenario == "read error" {
				wantErr = failure
				store.get = func(string) (string, bool, error) { return "", false, failure }
			} else if scenario == "insert error" || scenario == "insert race" {
				if scenario == "insert error" {
					wantErr = failure
				}
				store.putIfAbsent = func(key, value string) (bool, error) {
					require.Equal(t, "shared", key)
					require.Equal(t, locked, value)
					return false, wantErr
				}
			} else {
				require.NoError(t, store.Put("shared", unlocked))
				if scenario == "swap error" {
					wantErr = failure
				}
				store.compareAndSwap = func(key, previous, next string) (bool, error) {
					require.Equal(t, "shared", key)
					require.Equal(t, unlocked, previous)
					require.Equal(t, locked, next)
					return false, wantErr
				}
			}
			lock, ok, err := NewLocker(store).Lock("shared")
			require.ErrorIs(t, err, wantErr)
			require.False(t, ok)
			require.Nil(t, lock)
		})
	}
}

func TestUnlockStorageFailureCanBeRetried(t *testing.T) {
	store := &lockerStorageHooks{AtomicStorage: NewMemStorage()}
	locker := NewLocker(store)
	lock, ok, err := locker.Lock("shared")
	require.NoError(t, err)
	require.True(t, ok)
	failure := errors.New("unlock storage unavailable")
	store.compareAndSwap = func(key, previous, next string) (bool, error) {
		require.Equal(t, "shared", key)
		require.Equal(t, locked, previous)
		require.Equal(t, unlocked, next)
		return false, failure
	}
	require.ErrorIs(t, lock.Unlock(), failure)
	_, ok, err = locker.Lock("shared")
	require.NoError(t, err)
	require.False(t, ok)
	store.compareAndSwap = nil
	require.NoError(t, lock.Unlock())
	value, present, err := store.Get("shared")
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, unlocked, value)
}
