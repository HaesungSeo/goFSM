package fsmerror

import "errors"

var (
	ErrDupHandle       = errors.New("conflict handle")
	ErrInvalidEvent    = errors.New("invalid event")
	ErrHandleNotExists = errors.New("handle not exists")
)
