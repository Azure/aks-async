package operationsbus

import (
	"context"
)

// OpInterface is the interface all operations will need to implement.
type APIOperation interface {
	Run(ctx context.Context) *Result
	Guardconcurrency() (*CategorizedError, error)
	EntityFetcher() *Entity
}
