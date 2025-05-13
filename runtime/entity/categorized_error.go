package entity

// This is the error that should be returned by the concurrency check should anything go wrong. This allows us to provide more details on what happened.
// Mainly implemented to be usable by aks-rp users since they use this error type.
var _ error = &CategorizedError{}

type CategorizedError struct {
	Message      string
	InnerMessage string
	ErrorCode    int
	Err          error
}

func NewCategorizedError(message string, innerMessage string, errorCode int, err error) *CategorizedError {
	return &CategorizedError{
		Message:      message,
		InnerMessage: innerMessage,
		ErrorCode:    errorCode,
		Err:          err,
	}
}

func (ce *CategorizedError) Error() string {
	return ce.Message
}
