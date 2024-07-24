package operationsbus

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	//TODO(mheberling): figure out which general fields we need to add.
	OperationName string

	APIVersion string

	OperationId string

	// HTTP
	Body       []byte
	HttpMethod string
	RetryCount int
}
