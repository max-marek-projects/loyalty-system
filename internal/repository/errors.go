package repository

import "errors"

var ErrorUserNotFound = errors.New("User not found in storage")
var ErrorAlreadyInStorage = errors.New("Item already exists in storage")
var ErrorStorageConflict = errors.New("Current data conflicts with other record")
