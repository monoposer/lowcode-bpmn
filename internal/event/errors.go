package event

import "errors"

var (
	ErrStreamDisabled = errors.New("event: stream consumer disabled")
	ErrUnknownStream  = errors.New("event: unknown stream")
)
