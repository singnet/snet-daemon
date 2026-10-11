package etcddb

import (
	"errors"
	"testing"

	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
)

func (suite *EtcdTestSuite) TestCompleteTransactionRejectsStaleSnapshot() {
	for _, change := range []string{"replace", "delete", "restore original value", "delete and recreate"} {
		suite.T().Run(change, func(t *testing.T) {
			client := suite.client
			key, other := t.Name()+"/condition", t.Name()+"/other"
			require.NoError(t, client.Put(key, "original"))
			require.NoError(t, client.Put(other, "untouched"))
			txn, err := client.StartTransaction([]string{key, other})
			require.NoError(t, err)

			// Interleave a write after the snapshot, before the attempted commit.
			want := storage.KeyValueData{Key: key, Value: "original", Present: true}
			switch change {
			case "replace":
				require.NoError(t, client.Put(key, "concurrent"))
				want.Value = "concurrent"
			case "delete":
				require.NoError(t, client.Delete(key))
				want.Value, want.Present = "", false
			case "restore original value":
				require.NoError(t, client.Put(key, "concurrent"))
				require.NoError(t, client.Put(key, "original"))
			case "delete and recreate":
				require.NoError(t, client.Delete(key))
				require.NoError(t, client.Put(key, "original"))
			}

			updates := []storage.KeyValueData{
				{Key: key, Value: "committed", Present: true},
				{Key: other, Value: "committed", Present: true},
			}
			ok, err := client.CompleteTransaction(txn, updates)
			require.NoError(t, err)
			require.False(t, ok, "a changed revision must reject the entire transaction")
			value, present, err := client.Get(key)
			require.NoError(t, err)
			require.Equal(t, want.Value, value)
			require.Equal(t, want.Present, present)
			value, present, err = client.Get(other)
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, "untouched", value, "no partial writes on conflict")

			values, err := txn.GetConditionValues()
			require.NoError(t, err)
			require.ElementsMatch(t, []storage.KeyValueData{
				want, {Key: other, Value: "untouched", Present: true},
			}, values, "a rejected transaction must refresh its snapshot")

			ok, err = client.CompleteTransaction(txn, updates)
			require.NoError(t, err)
			require.True(t, ok, "the refreshed snapshot should allow a retry")
			for _, update := range updates {
				value, present, err := client.Get(update.Key)
				require.NoError(t, err)
				require.True(t, present)
				require.Equal(t, update.Value, value)
			}
		})
	}
}

func (suite *EtcdTestSuite) TestExecuteTransactionRetryPolicy() {
	for _, retry := range []bool{false, true} {
		name := "without retry"
		if retry {
			name = "with retry"
		}
		suite.T().Run(name, func(t *testing.T) {
			client := suite.client
			key := t.Name()
			require.NoError(t, client.Put(key, "original"))
			calls := 0
			ok, err := client.ExecuteTransaction(storage.CASRequest{
				ConditionKeys: []string{key}, RetryTillSuccessOrError: retry,
				Update: func(values []storage.KeyValueData) ([]storage.KeyValueData, bool, error) {
					calls++
					require.LessOrEqual(t, calls, 2, "retry must converge after the single conflicting write")
					want := "original"
					if calls == 2 {
						want = "concurrent"
					}
					require.Equal(t, []storage.KeyValueData{{Key: key, Value: want, Present: true}}, values)
					if calls == 1 {
						require.NoError(t, client.Put(key, "concurrent"))
					}
					return []storage.KeyValueData{{Key: key, Value: values[0].Value + " updated", Present: true}}, true, nil
				},
			})
			require.NoError(t, err)
			require.Equal(t, retry, ok)
			want, wantCalls := "concurrent", 1
			if retry {
				want, wantCalls = "concurrent updated", 2
			}
			require.Equal(t, wantCalls, calls)
			value, present, err := client.Get(key)
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, want, value)
		})
	}
}

func (suite *EtcdTestSuite) TestExecuteTransactionCallbackStopsRetries() {
	sentinel := errors.New("update rejected")
	for _, callbackErr := range []error{nil, sentinel} {
		name := "declined"
		if callbackErr != nil {
			name = "error"
		}
		suite.T().Run(name, func(t *testing.T) {
			key := t.Name()
			require.NoError(t, suite.client.Put(key, "original"))
			calls := 0
			ok, err := suite.client.ExecuteTransaction(storage.CASRequest{
				ConditionKeys: []string{key}, RetryTillSuccessOrError: true,
				Update: func([]storage.KeyValueData) ([]storage.KeyValueData, bool, error) {
					calls++
					require.Equal(t, 1, calls, "callback rejection must not trigger a retry")
					return []storage.KeyValueData{{Key: key, Value: "unexpected"}}, callbackErr != nil, callbackErr
				},
			})
			require.ErrorIs(t, err, callbackErr)
			require.False(t, ok)
			require.Equal(t, 1, calls)
			value, present, err := suite.client.Get(key)
			require.NoError(t, err)
			require.True(t, present)
			require.Equal(t, "original", value)
		})
	}
}

func TestCompleteTransactionRejectsForeignTransaction(t *testing.T) {
	client := &EtcdClient{}
	for _, txn := range []storage.Transaction{nil, &foreignTransaction{}} {
		ok, err := client.CompleteTransaction(txn, nil)
		require.ErrorContains(t, err, "unexpected transaction type")
		require.False(t, ok)
	}
}

type foreignTransaction struct{ storage.Transaction }

func TestTransactionConditionValuesAreIndependent(t *testing.T) {
	txn := &etcdTransaction{ConditionValues: []keyValueVersion{
		{Key: "present", Value: "", Present: true, Version: 42},
		{Key: "absent", Present: false},
	}}
	values, err := txn.GetConditionValues()
	require.NoError(t, err)
	want := []storage.KeyValueData{{Key: "present", Present: true}, {Key: "absent"}}
	require.Equal(t, want, values)
	values[0] = storage.KeyValueData{Key: "modified", Value: "modified"}
	again, err := txn.GetConditionValues()
	require.NoError(t, err)
	require.Equal(t, want, again, "callers must not be able to alter the stored snapshot")
}
