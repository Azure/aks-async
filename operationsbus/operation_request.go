package operationsbus

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	OperationName string
	APIVersion    string
	RetryCount    int
	OperationId   string

	// HTTP
	Body       []byte
	HttpMethod string
}
