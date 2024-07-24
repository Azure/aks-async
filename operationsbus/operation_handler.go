package operationsbus

import (
	"context"
)

// OpInterface is the interface all operations will need to implement.
type APIOperation interface {
	Run(ctx context.Context) *Result
	Retry(ctx context.Context) error
	Guardconcurrency(context.Context, *Entity) (*CategorizedError, error)
	EntityFetcher(context.Context) (*Entity, error)
	Init(context.Context, OperationRequest) (*APIOperation, error) //TODO(mheberling): Missing ctx here? and in all functions.
	GetName(context.Context) string
}
