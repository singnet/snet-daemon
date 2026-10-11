package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestCompleteTransactionConflictDoesNotPartiallyWrite(t *testing.T) {
	storage := NewMemStorage()
	require.NoError(t, storage.Put("first", "original-first"))
	require.NoError(t, storage.Put("second", "original-second"))
	transaction, err := storage.StartTransaction([]string{"first", "second"})
	require.NoError(t, err)

	// Another writer changes the last condition before this transaction commits.
	require.NoError(t, storage.Put("second", "concurrent"))
	ok, err := storage.CompleteTransaction(transaction, []KeyValueData{
		{Key: "first", Value: "updated-first", Present: true},
		{Key: "second", Value: "updated-second", Present: true},
	})
	require.NoError(t, err)
	require.False(t, ok)

	for key, want := range map[string]string{"first": "original-first", "second": "concurrent"} {
		value, present, err := storage.Get(key)
		require.NoError(t, err)
		require.True(t, present)
		require.Equal(t, want, value, "a rejected transaction must not write key %q", key)
	}
}

func TestCompleteTransactionRejectsChangedConditionOnlyKey(t *testing.T) {
	for _, change := range []string{"inserted", "modified", "deleted"} {
		t.Run(change, func(t *testing.T) {
			storage := NewMemStorage()
			require.NoError(t, storage.Put("target", "original"))
			if change != "inserted" {
				require.NoError(t, storage.Put("guard", "original"))
			}
			transaction, err := storage.StartTransaction([]string{"target", "guard"})
			require.NoError(t, err)
			if change == "deleted" {
				require.NoError(t, storage.Delete("guard"))
			} else {
				// An empty value must still count as a present key.
				require.NoError(t, storage.Put("guard", ""))
			}
			ok, err := storage.CompleteTransaction(transaction, []KeyValueData{
				{Key: "target", Value: "updated", Present: true},
			})
			require.NoError(t, err)
			require.False(t, ok)
			value, present, err := storage.Get("target")
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, "original", value)
		})
	}
}

func TestCompleteTransactionCompetingCommitsHaveOneWinner(t *testing.T) {
	for _, initiallyPresent := range []bool{false, true} {
		t.Run(fmt.Sprintf("initially_present_%t", initiallyPresent), func(t *testing.T) {
			for range 32 {
				storage := NewMemStorage()
				keys := []string{"first", "second"}
				if initiallyPresent {
					for _, key := range keys {
						require.NoError(t, storage.Put(key, "original"))
					}
				}
				transactions := make([]Transaction, 2)
				for i := range transactions {
					var err error
					transactions[i], err = storage.StartTransaction(keys)
					require.NoError(t, err)
				}
				type result struct {
					value string
					ok    bool
					err   error
				}
				start := make(chan struct{})
				results := make(chan result, 2)
				for i, transaction := range transactions {
					go func() {
						<-start
						value := fmt.Sprintf("writer-%d", i)
						ok, err := storage.CompleteTransaction(transaction, []KeyValueData{
							{Key: "first", Value: value, Present: true},
							{Key: "second", Value: value, Present: true},
						})
						results <- result{value: value, ok: ok, err: err}
					}()
				}
				close(start)
				winners := 0
				var winner string
				for range transactions {
					result := <-results
					require.NoError(t, result.err)
					if result.ok {
						winners++
						winner = result.value
					}
				}
				require.Equal(t, 1, winners)
				for _, key := range keys {
					value, present, err := storage.Get(key)
					require.NoError(t, err)
					require.True(t, present)
					require.Equal(t, winner, value)
				}
			}
		})
	}
}

func TestStartTransactionSeesConsistentSnapshotDuringCommits(t *testing.T) {
	storage := NewMemStorage()
	keys := []string{"first", "second"}
	for _, key := range keys {
		require.NoError(t, storage.Put(key, "original"))
	}
	const iterations = 1000
	start := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		<-start
		for i := range iterations {
			value := fmt.Sprintf("version-%d", i)
			ok, err := storage.ExecuteTransaction(CASRequest{
				ConditionKeys: keys,
				Update: func([]KeyValueData) ([]KeyValueData, bool, error) {
					return []KeyValueData{
						{Key: "first", Value: value, Present: true},
						{Key: "second", Value: value, Present: true},
					}, true, nil
				},
			})
			if err != nil || !ok {
				done <- fmt.Errorf("single writer commit failed: ok=%t, err=%v", ok, err)
				return
			}
		}
		done <- nil
	}()
	close(start)
	for range iterations {
		transaction, err := storage.StartTransaction(keys)
		require.NoError(t, err)
		values, err := transaction.GetConditionValues()
		require.NoError(t, err)
		require.Len(t, values, 2)
		require.True(t, values[0].Present)
		require.True(t, values[1].Present)
		require.Equal(t, values[0].Value, values[1].Value, "snapshot must not mix two commits")
	}
	require.NoError(t, <-done)
}

func TestExecuteTransactionLimitsAttemptsOnPersistentConflict(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprintf("retry_%t", retry), func(t *testing.T) {
			storage := NewMemStorage()
			require.NoError(t, storage.Put("key", "original"))
			attempts := 0
			var concurrentValue string
			ok, err := storage.ExecuteTransaction(CASRequest{
				ConditionKeys:           []string{"key"},
				RetryTillSuccessOrError: retry,
				Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
					attempts++
					concurrentValue = values[0].Value + "-concurrent"
					require.NoError(t, storage.Put("key", concurrentValue))
					return []KeyValueData{{Key: "key", Value: "updated", Present: true}}, true, nil
				},
			})
			require.NoError(t, err)
			require.False(t, ok)
			expectedAttempts := 1
			if retry {
				expectedAttempts = 100
			}
			require.Equal(t, expectedAttempts, attempts)
			value, present, err := storage.Get("key")
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, concurrentValue, value)
		})
	}
}

func TestExecuteTransactionRefreshesSnapshotAfterUpdateDeclines(t *testing.T) {
	storage := NewMemStorage()
	require.NoError(t, storage.Put("key", "original"))
	attempts := 0
	ok, err := storage.ExecuteTransaction(CASRequest{
		ConditionKeys:           []string{"key"},
		RetryTillSuccessOrError: true,
		Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
			attempts++
			if attempts == 1 {
				require.NoError(t, storage.Put("key", "concurrent"))
				return nil, false, nil
			}
			require.Equal(t, "concurrent", values[0].Value)
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
}

func TestExecuteTransactionRefreshesSnapshotAfterConflict(t *testing.T) {
	storage := NewMemStorage()
	require.NoError(t, storage.Put("key", "original"))
	var snapshots []string
	ok, err := storage.ExecuteTransaction(CASRequest{
		ConditionKeys:           []string{"key"},
		RetryTillSuccessOrError: true,
		Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
			require.Len(t, values, 1)
			require.True(t, values[0].Present)
			snapshots = append(snapshots, values[0].Value)
			if len(snapshots) == 1 {
				// One competing write; subsequent attempts have no contention.
				require.NoError(t, storage.Put("key", "concurrent"))
			}
			return []KeyValueData{{Key: "key", Value: values[0].Value + "-updated", Present: true}}, true, nil
		},
	})
	require.NoError(t, err)
	require.True(t, ok, "a single conflict must not exhaust all retries; attempts: %d", len(snapshots))
	require.Equal(t, []string{"original", "concurrent"}, snapshots)
	value, present, err := storage.Get("key")
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "concurrent-updated", value)
}

func TestExecuteTransaction_NoRetry(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("x", "1")

	req := CASRequest{
		ConditionKeys:           []string{"x"},
		RetryTillSuccessOrError: false,
		Update: func(old []KeyValueData) ([]KeyValueData, bool, error) {
			return []KeyValueData{{Key: "x", Value: "2", Present: true}}, false, nil
		},
	}

	ok, err := s.ExecuteTransaction(req)
	assert.NoError(t, err)
	assert.False(t, ok, "Transaction should fail without retry")
}

func TestExecuteTransaction_SuccessFirstTry(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("x", "1")

	req := CASRequest{
		ConditionKeys:           []string{"x"},
		RetryTillSuccessOrError: false,
		Update: func(old []KeyValueData) ([]KeyValueData, bool, error) {
			return []KeyValueData{{Key: "x", Value: "2", Present: true}}, true, nil
		},
	}

	ok, err := s.ExecuteTransaction(req)
	assert.NoError(t, err)
	assert.True(t, ok, "Transaction should succeed")

	value, _, _ := s.Get("x")
	assert.Equal(t, "2", value, "Value should be updated to '2'")
}

func TestExecuteTransaction_RetryUntilSuccess(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("x", "1")

	attempts := 0
	req := CASRequest{
		ConditionKeys:           []string{"x"},
		RetryTillSuccessOrError: true,
		Update: func(old []KeyValueData) ([]KeyValueData, bool, error) {
			attempts++
			if attempts < 3 {
				return []KeyValueData{{Key: "x", Value: "999", Present: true}}, false, nil
			}
			return []KeyValueData{{Key: "x", Value: "42", Present: true}}, true, nil
		},
	}

	ok, err := s.ExecuteTransaction(req)
	assert.NoError(t, err)
	assert.True(t, ok, "Transaction should eventually succeed")
	assert.Equal(t, 3, attempts, "Expected 3 attempts")

	value, _, _ := s.Get("x")
	assert.Equal(t, "42", value, "Value should be updated to '42'")
}

func TestExecuteTransaction_RetryFails(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("x", "1")

	attempts := 0
	req := CASRequest{
		ConditionKeys:           []string{"x"},
		RetryTillSuccessOrError: true,
		Update: func(old []KeyValueData) ([]KeyValueData, bool, error) {
			attempts++
			return []KeyValueData{{Key: "x", Value: "2", Present: true}}, false, nil
		},
	}

	ok, err := s.ExecuteTransaction(req)
	assert.NoError(t, err)
	assert.False(t, ok, "Transaction should fail after retries")
	assert.Greater(t, attempts, 0, "Expected at least one attempt")

	value, _, _ := s.Get("x")
	assert.Equal(t, "1", value, "Value should remain unchanged")
}

func TestExecuteTransaction_Retry(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("x", "1")
	attempts := 0

	req := CASRequest{
		ConditionKeys:           []string{"x"},
		RetryTillSuccessOrError: true,
		Update: func(old []KeyValueData) ([]KeyValueData, bool, error) {
			attempts++
			if attempts == 1 {
				return []KeyValueData{{Key: "x", Value: "wrong", Present: true}}, false, nil
			}
			return []KeyValueData{{Key: "x", Value: "2", Present: true}}, true, nil
		},
	}

	ok, err := s.ExecuteTransaction(req)
	assert.NoError(t, err)
	assert.True(t, ok, "Transaction should succeed after retry")
}
