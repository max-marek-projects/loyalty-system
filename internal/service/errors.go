package service

import (
	"errors"
)

var ErrorLoginAlreadyTaken = errors.New("login already taken by other user")
var ErrorWrongUsernamePassword = errors.New("wrong username or password")
var ErrorOrdersConflict = errors.New("orders conflicts with other user")
var ErrorNumberNotValid = errors.New("order number not valid by Luhn algorithm")
var ErrInsufficientFunds = errors.New("user has insufficient funds")

// external service
var ErrorOrderNotYetProcessed = errors.New("order not yet processed")
