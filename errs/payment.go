package errs

import "fmt"

// PaymentErrorCode contains all types of errors which we need to handle on the
// client side.
type PaymentErrorCode int

const (
	// Internal error code means that error is caused by improper daemon
	// configuration or functioning. Client cannot do anything with it.
	Internal PaymentErrorCode = 1
	// Unauthenticated error code means that client sent payment which cannot
	// be applied to the channel.
	Unauthenticated PaymentErrorCode = 2
	// FailedPrecondition means that request cannot be handled because system
	// is not in appropriate state.
	FailedPrecondition PaymentErrorCode = 3
	// IncorrectNonce is returned when nonce value sent by client is incorrect.
	IncorrectNonce PaymentErrorCode = 4
)

// PaymentError contains error code and message and implements Error interface.
type PaymentError struct {
	// Code is error code
	Code PaymentErrorCode
	// Message is message
	Message string
}

// NewPaymentError constructs new PaymentError instance with given error code
// and message.
func NewPaymentError(code PaymentErrorCode, format string, msg ...any) *PaymentError {
	if len(msg) == 0 {
		return &PaymentError{Code: code, Message: format}
	}
	return &PaymentError{Code: code, Message: fmt.Sprintf(format, msg...)}
}

func (err *PaymentError) Error() string {
	return err.Message
}
