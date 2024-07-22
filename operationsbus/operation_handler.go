package operationsbus

import (
	"context"
)

// OpInterface is the interface all operations will need to implement.
type APIOperation interface {
	Run(ctx context.Context) *Result
	Retry(ctx context.Context) error
	Guardconcurrency(*Entity) (*CategorizedError, error)
	EntityFetcher() (*Entity, error)
	Init(OperationRequest) (*APIOperation, error)
	GetName() string
}
