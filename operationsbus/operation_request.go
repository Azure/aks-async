package operationsbus

import (
	"context"
)

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	//TODO(mheberling): figure out which general fields we need to add.
	APIVersion string

	Context context.Context

	HttpMethod  string
	OperationId string

	// HTTP
	Body []byte
}
