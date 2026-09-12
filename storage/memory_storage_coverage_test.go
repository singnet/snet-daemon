package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartTransactionCapturesPresentAndMissingKeys(t *testing.T) {
	storage := NewMemStorage()
	require.NoError(t, storage.Put("present", "value"))

	transaction, err := storage.StartTransaction([]string{"present", "missing"})
	require.NoError(t, err)
	values, err := transaction.GetConditionValues()
	require.NoError(t, err)
	require.Equal(t, []KeyValueData{
		{Key: "present", Value: "value", Present: true},
		{Key: "missing", Value: "", Present: false},
	}, values)

	values[0].Value = "mutated copy"
	secondRead, err := transaction.GetConditionValues()
	require.NoError(t, err)
	require.Equal(t, "value", secondRead[0].Value)
}

func TestGetValueDataForKeyReturnsNotFound(t *testing.T) {
	data, present := getValueDataForKey("missing", []KeyValueData{{Key: "other", Value: "value"}})

	require.False(t, present)
	require.Equal(t, KeyValueData{}, data)
}

func TestCompleteTransactionCreatesPreviouslyMissingKey(t *testing.T) {
	storage := NewMemStorage()
	transaction, err := storage.StartTransaction([]string{"missing"})
	require.NoError(t, err)

	ok, err := storage.CompleteTransaction(transaction, []KeyValueData{{Key: "missing", Value: "created", Present: true}})
	require.NoError(t, err)
	require.True(t, ok)
	value, present, err := storage.Get("missing")
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "created", value)
}

func TestCompleteTransactionDetectsConcurrentChanges(t *testing.T) {
	t.Run("previously missing key was inserted", func(t *testing.T) {
		storage := NewMemStorage()
		transaction, err := storage.StartTransaction([]string{"key"})
		require.NoError(t, err)
		require.NoError(t, storage.Put("key", "concurrent"))

		ok, err := storage.CompleteTransaction(transaction, []KeyValueData{{Key: "key", Value: "updated", Present: true}})
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("existing key was deleted", func(t *testing.T) {
		storage := NewMemStorage()
		require.NoError(t, storage.Put("key", "original"))
		transaction, err := storage.StartTransaction([]string{"key"})
		require.NoError(t, err)
		require.NoError(t, storage.Delete("key"))

		ok, err := storage.CompleteTransaction(transaction, []KeyValueData{{Key: "key", Value: "updated", Present: true}})
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("existing key was modified", func(t *testing.T) {
		storage := NewMemStorage()
		require.NoError(t, storage.Put("key", "original"))
		transaction, err := storage.StartTransaction([]string{"key"})
		require.NoError(t, err)
		require.NoError(t, storage.Put("key", "concurrent"))

		ok, err := storage.CompleteTransaction(transaction, []KeyValueData{{Key: "key", Value: "updated", Present: true}})
		require.NoError(t, err)
		require.False(t, ok)
		value, _, err := storage.Get("key")
		require.NoError(t, err)
		require.Equal(t, "concurrent", value)
	})
}

func TestCompleteTransactionAcceptsConditionOnlyKeys(t *testing.T) {
	storage := NewMemStorage()
	require.NoError(t, storage.Put("present", "value"))
	transaction, err := storage.StartTransaction([]string{"present", "missing"})
	require.NoError(t, err)

	ok, err := storage.CompleteTransaction(transaction, nil)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestExecuteTransactionReturnsUpdateError(t *testing.T) {
	storage := NewMemStorage()
	expected := errors.New("update failed")

	ok, err := storage.ExecuteTransaction(CASRequest{
		ConditionKeys: []string{"key"},
		Update: func([]KeyValueData) ([]KeyValueData, bool, error) {
			return nil, false, expected
		},
	})

	require.False(t, ok)
	require.ErrorIs(t, err, expected)
}

func TestExecuteTransactionHandlesCompletionConflict(t *testing.T) {
	t.Run("without retry", func(t *testing.T) {
		storage := NewMemStorage()
		ok, err := storage.ExecuteTransaction(CASRequest{
			ConditionKeys: []string{"key"},
			Update: func([]KeyValueData) ([]KeyValueData, bool, error) {
				require.NoError(t, storage.Put("key", "concurrent"))
				return []KeyValueData{{Key: "key", Value: "updated", Present: true}}, true, nil
			},
		})

		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("retry succeeds", func(t *testing.T) {
		storage := NewMemStorage()
		attempts := 0
		ok, err := storage.ExecuteTransaction(CASRequest{
			ConditionKeys:           []string{"key"},
			RetryTillSuccessOrError: true,
			Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
				attempts++
				if attempts == 1 {
					require.False(t, values[0].Present)
					require.NoError(t, storage.Put("key", "concurrent"))
				} else {
					require.True(t, values[0].Present)
					require.Equal(t, "concurrent", values[0].Value)
				}
				return []KeyValueData{{Key: "key", Value: "updated", Present: true}}, true, nil
			},
		})

		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, 2, attempts)
		value, present, err := storage.Get("key")
		require.NoError(t, err)
		require.True(t, present)
		require.Equal(t, "updated", value)
	})
}
