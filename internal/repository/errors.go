package repository

import (
	"errors"
)

// ErrUserNotFound is returned when a user login does not exist.
var ErrUserNotFound = errors.New("user not found in storage")

// ErrAlreadyInStorage is returned when attempting to insert a duplicate record.
var ErrAlreadyInStorage = errors.New("item already exists in storage")

// ErrStorageConflict is returned when an order belongs to another user.
var ErrStorageConflict = errors.New("current data conflicts with other record")

// ErrInsufficientFunds is returned when a withdrawal exceeds available balance.
var ErrInsufficientFunds = errors.New("user has insufficient funds")
