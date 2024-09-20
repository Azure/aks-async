package operationsbus

import (
	"context"
)

// APIOperation is the interface all operations will need to implement.
type APIOperation interface {
	Init(context.Context, OperationRequest) (APIOperation, error)
	GuardConcurrency(context.Context, Entity) (*CategorizedError, error)
	Run(context.Context) *Result
}
