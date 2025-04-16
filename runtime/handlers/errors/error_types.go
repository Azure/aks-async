package errors

import (
	"fmt"
)

// Default errors for the error handler.
type RetryError struct {
	Message string
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("RetryError: %s", e.Message)
}

type NonRetryError struct {
	Message string
}

func (e *NonRetryError) Error() string {
	return fmt.Sprintf("NonRetryError: %s", e.Message)
}
