package service

import (
	"errors"
	"fmt"
	"time"
)

// ErrLoginAlreadyTaken is returned when registration login already exists.
var ErrLoginAlreadyTaken = errors.New("login already taken by other user")

// ErrWrongUsernamePassword is returned when login credentials are invalid.
var ErrWrongUsernamePassword = errors.New("wrong username or password")

// ErrOrdersConflict is returned when an order belongs to another user.
var ErrOrdersConflict = errors.New("orders conflicts with other user")

// ErrNumberNotValid is returned when order number fails Luhn validation.
var ErrNumberNotValid = errors.New("order number not valid by Luhn algorithm")

// ErrInsufficientFunds is returned when withdrawal exceeds available balance.
var ErrInsufficientFunds = errors.New("user has insufficient funds")

// ErrOrderNotYetProcessed is returned when the accrual system indicates the order is not ready.
type ErrOrderNotYetProcessed struct {
	Err        error
	RetryAfter time.Duration
}

func (e *ErrOrderNotYetProcessed) Error() string {
	return fmt.Sprintf("Order not yet processed: %v. Retry after: %v", e.Err, e.RetryAfter)
}

func (e *ErrOrderNotYetProcessed) Unwrap() error {
	return e.Err
}
