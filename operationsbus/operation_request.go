package operationsbus

import "google.golang.org/protobuf/types/known/timestamppb"

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	OperationName  string
	APIVersion     string
	RetryCount     int
	OperationId    string
	EntityId       string
	EntityType     string
	ExpirationDate *timestamppb.Timestamp

	// HTTP
	Body       []byte
	HttpMethod string
}
