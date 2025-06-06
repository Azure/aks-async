package errors

import (
	"fmt"
)

type NonRetryError struct {
	Message string
}

func (e *NonRetryError) Error() string {
	return fmt.Sprintf("NonRetryError: %s", e.Message)
}
