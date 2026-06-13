package service

import (
	"errors"
	"fmt"
	"time"
)

// ErrorLoginAlreadyTaken is returned when registration login already exists.
var ErrorLoginAlreadyTaken = errors.New("login already taken by other user")

// ErrorWrongUsernamePassword is returned when login credentials are invalid.
var ErrorWrongUsernamePassword = errors.New("wrong username or password")

// ErrorOrdersConflict is returned when an order belongs to another user.
var ErrorOrdersConflict = errors.New("orders conflicts with other user")

// ErrorNumberNotValid is returned when order number fails Luhn validation.
var ErrorNumberNotValid = errors.New("order number not valid by Luhn algorithm")

// ErrInsufficientFunds is returned when withdrawal exceeds available balance.
var ErrInsufficientFunds = errors.New("user has insufficient funds")

// ErrorOrderNotYetProcessed is returned when the accrual system indicates the order is not ready.
type ErrorOrderNotYetProcessed struct {
	Err        error
	RetryAfter time.Duration
}

func (e *ErrorOrderNotYetProcessed) Error() string {
	return fmt.Sprintf("Order not yet processed: %v. Retry after: %v", e.Err, e.RetryAfter)
}

func (e *ErrorOrderNotYetProcessed) Unwrap() error {
	return e.Err
}
