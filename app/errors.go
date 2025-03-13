package app

import "errors"

var (
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrNotFound          = errors.New("record not found")
	ErrUnAuthorized      = errors.New("unauthorized")
	ErrInternal          = errors.New("internal error")
	ErrImageAlreadySaved = errors.New("image already in collection")
	ErrImageNotSaved     = errors.New("image not in collection")
)
