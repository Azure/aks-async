package operationsbus

// This is the error that should be return by the concurrency check should anything go wrong. This allows us to provide more details on what happened. aks-rp repo does the same thing.
var _ error = &CategorizedError{}

type CategorizedError struct {
	Message      string
	InnerMessage string
	ErrorCode    int
}

func NewCategorizedError(message string, innerMessage string, errorCode int) *CategorizedError {
	return &CategorizedError{
		Message:      message,
		InnerMessage: innerMessage,
		ErrorCode:    errorCode,
	}
}

func (ce *CategorizedError) Error() string {
	return ce.Message
}
