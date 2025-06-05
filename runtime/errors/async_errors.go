package errors

import (
	"fmt"
	"time"
)

type AsyncError struct {
	Message       string
	ErrorCode     int
	RetryAfter    time.Duration
	OriginalError error
}

func (e *AsyncError) Error() string {
	return fmt.Sprintf("AsyncError: OriginalError: %s", e.OriginalError.Error())
}

func (e *AsyncError) Unwrap() error {
	return e.OriginalError
}
