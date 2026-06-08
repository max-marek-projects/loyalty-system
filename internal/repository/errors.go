package repository

import (
	"errors"
)

var ErrUserNotFound = errors.New("User not found in storage")
var ErrAlreadyInStorage = errors.New("Item already exists in storage")
var ErrStorageConflict = errors.New("Current data conflicts with other record")
var ErrInsufficientFunds = errors.New("User has insufficient funds")
