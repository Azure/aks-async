package operationsbus

// This is the return value of the Run method, should we want to add some conditional logic depending on how the operation ended.
type Result struct {
	//TODO(mheberling): figure out which general fields we need to add. And how we want to use the Result.
	HTTPCode int
	Message  string
}

func NewResult(httpCode int, message string) *Result {
	return &Result{
		HTTPCode: httpCode,
		Message:  message,
	}
}
