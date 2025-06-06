package errors

import (
	"fmt"
)

type RetryError struct {
	Message string
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("RetryError: %s", e.Message)
}
