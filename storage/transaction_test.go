package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type transactionBackendStub struct {
	AtomicStorage
	start    func([]string) (Transaction, error)
	complete func(Transaction, []KeyValueData) (bool, error)
}

func (s *transactionBackendStub) StartTransaction(keys []string) (Transaction, error) {
	return s.start(keys)
}

func (s *transactionBackendStub) CompleteTransaction(tx Transaction, update []KeyValueData) (bool, error) {
	return s.complete(tx, update)
}

type transactionValuesStub struct {
	read func() ([]KeyValueData, error)
}

func (tx *transactionValuesStub) GetConditionValues() ([]KeyValueData, error) {
	return tx.read()
}

func TestExecuteTransactionStopsOnErrors(t *testing.T) {
	stages := []string{"start", "read", "update", "complete"}
	for index, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			expected := errors.New(stage + " failed")
			var calls []string
			record := func(current string) error {
				calls = append(calls, current)
				if current == stage {
					return expected
				}
				return nil
			}
			backend := &transactionBackendStub{
				start: func([]string) (Transaction, error) {
					return &transactionValuesStub{read: func() ([]KeyValueData, error) {
						return nil, record("read")
					}}, record("start")
				},
				complete: func(Transaction, []KeyValueData) (bool, error) {
					return true, record("complete")
				},
			}
			ok, err := ExecuteTransaction(backend, CASRequest{
				RetryTillSuccessOrError: true,
				Update: func([]KeyValueData) ([]KeyValueData, bool, error) {
					return nil, true, record("update")
				},
			}, TransactionPolicy{MaxAttempts: 3, RetryOnUpdateDeclined: true, RestartTransactionOnRetry: true})

			require.False(t, ok)
			require.ErrorIs(t, err, expected)
			require.Equal(t, stages[:index+1], calls, "errors must stop execution without retrying")
		})
	}
}

func TestExecuteTransactionPolicy(t *testing.T) {
	tests := []struct {
		name        string
		policy      TransactionPolicy
		retry       bool
		declines    int
		conflicts   int
		wantUpdates int
		wantCommits int
		wantStarts  int
		wantOK      bool
	}{
		{
			name:        "success without retry",
			wantUpdates: 1, wantCommits: 1, wantStarts: 1, wantOK: true,
		},
		{
			name:      "request disables retry on conflict",
			policy:    TransactionPolicy{MaxAttempts: 3, RestartTransactionOnRetry: true},
			conflicts: 1, wantUpdates: 1, wantCommits: 1, wantStarts: 1,
		},
		{
			name:   "success on last allowed attempt",
			policy: TransactionPolicy{MaxAttempts: 3, RestartTransactionOnRetry: true},
			retry:  true, conflicts: 2,
			wantUpdates: 3, wantCommits: 3, wantStarts: 3, wantOK: true,
		},
		{
			name:   "conflicts exhaust attempt limit",
			policy: TransactionPolicy{MaxAttempts: 2, RestartTransactionOnRetry: true},
			retry:  true, conflicts: 3,
			wantUpdates: 2, wantCommits: 2, wantStarts: 2,
		},
		{
			name:  "zero policy has no attempt limit",
			retry: true, conflicts: 101,
			wantUpdates: 102, wantCommits: 102, wantStarts: 1, wantOK: true,
		},
		{
			name:   "negative limit is unlimited",
			policy: TransactionPolicy{MaxAttempts: -1}, retry: true, conflicts: 2,
			wantUpdates: 3, wantCommits: 3, wantStarts: 1, wantOK: true,
		},
		{
			name:  "zero policy stops on declined update",
			retry: true, declines: 1, wantUpdates: 1, wantStarts: 1,
		},
		{
			name:     "request disables retry on declined update",
			policy:   TransactionPolicy{MaxAttempts: 3, RetryOnUpdateDeclined: true, RestartTransactionOnRetry: true},
			declines: 1, wantUpdates: 1, wantStarts: 1,
		},
		{
			name:   "declines and conflicts share attempt limit",
			policy: TransactionPolicy{MaxAttempts: 3, RetryOnUpdateDeclined: true, RestartTransactionOnRetry: true},
			retry:  true, declines: 1, conflicts: 1,
			wantUpdates: 3, wantCommits: 2, wantStarts: 3, wantOK: true,
		},
		{
			name:   "declined updates exhaust attempt limit without committing",
			policy: TransactionPolicy{MaxAttempts: 2, RetryOnUpdateDeclined: true, RestartTransactionOnRetry: true},
			retry:  true, declines: 3, wantUpdates: 2, wantStarts: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			starts, updates, commits := 0, 0, 0
			conditionValues := []KeyValueData{{Key: "key", Value: "original", Present: true}}
			newValues := []KeyValueData{{Key: "key", Value: "updated", Present: true}}
			backend := &transactionBackendStub{
				start: func(keys []string) (Transaction, error) {
					starts++
					require.Equal(t, []string{"key"}, keys)
					return &transactionValuesStub{read: func() ([]KeyValueData, error) {
						return conditionValues, nil
					}}, nil
				},
				complete: func(_ Transaction, values []KeyValueData) (bool, error) {
					commits++
					require.Equal(t, newValues, values)
					return commits > tt.conflicts, nil
				},
			}
			ok, err := ExecuteTransaction(backend, CASRequest{
				ConditionKeys: []string{"key"}, RetryTillSuccessOrError: tt.retry,
				Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
					updates++
					require.LessOrEqual(t, updates, tt.wantUpdates, "execution must terminate within the expected attempts")
					require.Equal(t, conditionValues, values)
					return newValues, updates > tt.declines, nil
				},
			}, tt.policy)

			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantUpdates, updates)
			require.Equal(t, tt.wantCommits, commits)
			require.Equal(t, tt.wantStarts, starts)
		})
	}
}

func TestExecuteTransactionReusesRefreshedSnapshot(t *testing.T) {
	values := []KeyValueData{{Key: "key", Value: "original", Present: true}}
	tx := &transactionValuesStub{read: func() ([]KeyValueData, error) {
		return values, nil
	}}
	starts, commits := 0, 0
	backend := &transactionBackendStub{
		start: func([]string) (Transaction, error) {
			starts++
			return tx, nil
		},
		complete: func(transaction Transaction, update []KeyValueData) (bool, error) {
			commits++
			require.Same(t, tx, transaction)
			if commits == 1 {
				require.Equal(t, "original updated", update[0].Value)
				values = []KeyValueData{{Key: "key", Value: "concurrent", Present: true}}
				return false, nil
			}
			require.Equal(t, "concurrent updated", update[0].Value)
			return true, nil
		},
	}
	var snapshots []string
	ok, err := ExecuteTransaction(backend, CASRequest{
		ConditionKeys: []string{"key"}, RetryTillSuccessOrError: true,
		Update: func(values []KeyValueData) ([]KeyValueData, bool, error) {
			snapshots = append(snapshots, values[0].Value)
			return []KeyValueData{{Key: "key", Value: values[0].Value + " updated", Present: true}}, true, nil
		},
	}, TransactionPolicy{})

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, starts, "a refreshed transaction must not be restarted")
	require.Equal(t, 2, commits)
	require.Equal(t, []string{"original", "concurrent"}, snapshots)
}
