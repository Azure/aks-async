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
	return fmt.Sprintf("AsyncError: Message: %s", e.Message)
}
