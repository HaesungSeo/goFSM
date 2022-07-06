package fsmerror

import "errors"

var (
	ErrDupHandle       = errors.New("conflict handle")
	ErrInvalidState    = errors.New("invalid state")
	ErrInvNextState    = errors.New("invalid next state")
	ErrInvalidEvent    = errors.New("invalid event")
	ErrHandleNotExists = errors.New("handle not exists")
)
