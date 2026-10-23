package storage

import "context"

// TransactionPolicy controls retries and snapshot handling in ExecuteTransaction.
type TransactionPolicy struct {
	// Context bounds retries; individual backend operations retain their own deadlines.
	Context context.Context
	// MaxAttempts includes the initial attempt. Non-positive values mean no limit.
	MaxAttempts int
	// RetryOnUpdateDeclined allows retrying when Update returns false without an error.
	// It only takes effect when CASRequest.RetryTillSuccessOrError is true.
	RetryOnUpdateDeclined bool
	// RestartTransactionOnRetry obtains a new snapshot before each retry.
	// Otherwise, CompleteTransaction must refresh the transaction after a conflict.
	RestartTransactionOnRetry bool
}

// ExecuteTransaction runs a CAS request using the backend's transaction primitives.
// Errors always stop execution. Conflicts are retried only when requested, subject
// to the policy's attempt limit. Rejection or exhaustion returns false without an error.
func ExecuteTransaction(backend AtomicStorage, request CASRequest, policy TransactionPolicy) (bool, error) {
	var transaction Transaction
	for attempt := 0; policy.MaxAttempts <= 0 || attempt < policy.MaxAttempts; attempt++ {
		if policy.Context != nil && policy.Context.Err() != nil {
			return false, policy.Context.Err()
		}
		if attempt == 0 || policy.RestartTransactionOnRetry {
			var err error
			transaction, err = backend.StartTransaction(request.ConditionKeys)
			if err != nil {
				return false, err
			}
		}

		oldValues, err := transaction.GetConditionValues()
		if err != nil {
			return false, err
		}
		newValues, ok, err := request.Update(oldValues)
		if err != nil {
			return false, err
		}
		if !ok {
			if !request.RetryTillSuccessOrError || !policy.RetryOnUpdateDeclined {
				return false, nil
			}
			continue
		}

		ok, err = backend.CompleteTransaction(transaction, newValues)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if !request.RetryTillSuccessOrError {
			return false, nil
		}
	}
	return false, nil
}
