package fsmerror

import "errors"

var (
	ErrState  = errors.New("invalid state")
	ErrEvent  = errors.New("invalid event")
	ErrHandle = errors.New("invalid handle")
)
