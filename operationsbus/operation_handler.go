package operationsbus

import (
	"context"
)

// OpInterface is the interface all operations will need to implement.
type APIOperation interface {
	Run(ctx context.Context) *Result
	Retry(ctx context.Context) error
	Guardconcurrency() (*CategorizedError, error)
	EntityFetcher() *Entity
	Init(OperationRequest) (*APIOperation, error)
	//TODO(mheberling): Add factory operation which will figure out which operation we're running.
}
