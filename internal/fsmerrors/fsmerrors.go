package fsmerror

import "errors"

var (
	ErrDupHandle       = errors.New("conflict handle")
	ErrInvalidState    = errors.New("invalid state")
	ErrInvNextState    = errors.New("invalid next state")
	ErrInvalidEvent    = errors.New("invalid event")
	ErrInvalidRetCode  = errors.New("invalid return code")
	ErrInvalidUserData = errors.New("invalid userdata")
	ErrHandleNotExists = errors.New("handle not exists")
	ErrHandleNoRetCode = errors.New("handle has no returncode")
)
