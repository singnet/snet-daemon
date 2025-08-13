package training

import (
	"errors"
	"fmt"
)

// Base Error
var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrUpdatingModel     = errors.New("error in updating model state")
	ErrServiceInvocation = errors.New("error in invoking service for model training")
	ErrServiceIssue      = errors.New("issue with service")
	ErrAccessToModel     = errors.New("unable to access model")
	ErrDaemonStorage     = errors.New("daemon storage error")
	ErrModelDoesntExist  = errors.New("model doesn't exist")
)

// Specific Error
var (
	ErrEmptyResponse             = fmt.Errorf("%w: service returned empty response", ErrServiceInvocation)
	ErrEmptyModelIDFromService   = fmt.Errorf("%w: service returned empty modelID", ErrServiceInvocation)
	ErrServiceIssueValidateModel = fmt.Errorf("%w: ValidateModel method error", ErrServiceIssue)
	ErrNoAuthorization           = fmt.Errorf("%w: no authorization provided", ErrInvalidRequest)
	ErrBadAuthorization          = fmt.Errorf("%w: bad authorization provided", ErrInvalidRequest)
	ErrNoGRPCServiceOrMethod     = fmt.Errorf("%w: no grpc_service_name or grpc_method_name provided", ErrInvalidRequest)
	ErrGetUserModelStorage       = fmt.Errorf("%w: error in getting data from user model storage", ErrDaemonStorage)
	ErrGetModelStorage           = fmt.Errorf("%w: error in getting data from model storage", ErrDaemonStorage)
	ErrPutModelStorage           = fmt.Errorf("%w: error in putting data to model storage", ErrDaemonStorage)
	ErrEmptyModelID              = fmt.Errorf("%w: model id can't be empty", ErrInvalidRequest)
	ErrNotOwnerModel             = fmt.Errorf("%w: only owner can change the model state", ErrUpdatingModel)
)

// WrapError formats and wraps an error with additional context.
func WrapError(baseErr error, message string) error {
	if baseErr == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", baseErr, message)
}
