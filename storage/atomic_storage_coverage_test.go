package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedStorageRejectsInvalidWritesWithoutChangingData(t *testing.T) {
	tests := []struct {
		name  string
		write func(*testing.T, TypedAtomicStorage) error
	}{
		{"put if absent invalid value", func(t *testing.T, s TypedAtomicStorage) error {
			ok, err := s.PutIfAbsent("key", "not-a-pointer")
			require.False(t, ok)
			return err
		}},
		{"compare and swap invalid key", func(t *testing.T, s TypedAtomicStorage) error {
			ok, err := s.CompareAndSwap(42, new("original"), new("updated"))
			require.False(t, ok)
			return err
		}},
		{"compare and swap invalid new value", func(t *testing.T, s TypedAtomicStorage) error {
			ok, err := s.CompareAndSwap("key", new("original"), "not-a-pointer")
			require.False(t, ok)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMemStorage()
			require.NoError(t, backend.Put("key", "original"))
			require.Error(t, tt.write(t, newStringTypedStorage(backend)))
			value, present, err := backend.Get("key")
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, "original", value)
		})
	}
}

func TestTypedStorageRejectsUnreadableValues(t *testing.T) {
	backend := NewMemStorage()
	require.NoError(t, backend.Put("key", "corrupt"))
	storage := newStringTypedStorage(backend).(*TypedAtomicStorageImpl)
	expected := errors.New("corrupt stored value")
	storage.valueDeserializer = func(string, any) error { return expected }

	value, ok, err := storage.Get("key")
	require.ErrorIs(t, err, expected)
	require.False(t, ok)
	require.Nil(t, value)

	values, err := storage.GetAll()
	require.ErrorIs(t, err, expected)
	require.Nil(t, values)

	ok, err = storage.ExecuteTransaction(TypedCASRequest{
		ConditionKeys: []any{"key"},
		Update: func([]TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			t.Fatal("update must not run with unreadable condition values")
			return nil, false, nil
		},
	})
	require.ErrorIs(t, err, expected)
	require.False(t, ok)
}

func TestTypedTransactionRejectsInvalidUpdatesWithoutWriting(t *testing.T) {
	expected := errors.New("update failed")
	tests := []struct {
		name       string
		conditions []any
		update     TypedUpdateFunc
	}{
		{"invalid condition key", []any{42}, nil},
		{"callback error", []any{"key"}, func([]TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			return []TypedKeyValueData{{Key: "key", Value: new("updated"), Present: true}}, true, expected
		}},
		{"invalid update key", []any{"key"}, func([]TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			return []TypedKeyValueData{{Key: 42, Value: new("updated"), Present: true}}, true, nil
		}},
		{"invalid update value", []any{"key"}, func([]TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			return []TypedKeyValueData{{Key: "key", Value: "not-a-pointer", Present: true}}, true, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMemStorage()
			require.NoError(t, backend.Put("key", "original"))
			ok, err := newStringTypedStorage(backend).ExecuteTransaction(TypedCASRequest{
				ConditionKeys: tt.conditions,
				Update: func(values []TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
					if tt.update == nil {
						t.Fatal("update must not run with invalid condition keys")
					}
					return tt.update(values)
				},
			})
			require.Error(t, err)
			if tt.name == "callback error" {
				require.ErrorIs(t, err, expected)
			}
			require.False(t, ok)
			value, present, err := backend.Get("key")
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, "original", value)
		})
	}
}

func TestTypedPrefixedTransactionWithNoUpdatesPreservesData(t *testing.T) {
	backend := NewMemStorage()
	require.NoError(t, backend.Put("prefix/key", "original"))
	storage := newStringTypedStorage(NewPrefixedAtomicStorage(backend, "prefix"))
	ok, err := storage.ExecuteTransaction(TypedCASRequest{
		ConditionKeys: []any{"key", "missing"},
		Update: func(values []TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			require.Equal(t, []TypedKeyValueData{
				{Key: "key", Value: new("original"), Present: true},
				{Key: "missing", Present: false},
			}, values)
			return nil, true, nil
		},
	})
	require.NoError(t, err)
	require.True(t, ok)
	values, err := backend.GetByKeyPrefix("")
	require.NoError(t, err)
	require.Equal(t, []string{"original"}, values)
}
