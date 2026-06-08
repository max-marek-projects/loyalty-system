package service

import (
	"errors"
)

var ErrorLoginAlreadyTaken = errors.New("Login already taken by other user")
var ErrorWrongUsernamePassword = errors.New("Wrong username or password")
var ErrorOrdersConflict = errors.New("Orders conflicts with other user")
var ErrorNumberNotValid = errors.New("Order number not valid by Luhn algorithm")
var ErrInsufficientFunds = errors.New("User has insufficient funds")

// external service
var ErrorOrderNotYetProcessed = errors.New("order not yet processed")
