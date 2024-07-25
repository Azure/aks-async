package operationsbus

// This is the return value of the Run method, should we want to add some conditional logic depending on how the operation ended.
type Result struct {
	HTTPCode int
	Message  string
	Error    error
}

func NewResult(httpCode int, message string, err error) *Result {
	return &Result{
		HTTPCode: httpCode,
		Message:  message,
		Error:    err,
	}
}
