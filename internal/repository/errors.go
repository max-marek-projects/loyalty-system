package repository

import (
	"errors"
)

var ErrUserNotFound = errors.New("user not found in storage")
var ErrAlreadyInStorage = errors.New("item already exists in storage")
var ErrStorageConflict = errors.New("current data conflicts with other record")
var ErrInsufficientFunds = errors.New("user has insufficient funds")
