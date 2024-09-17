package operationsbus

import (
	"context"
)

// APIOperation is the interface all operations will need to implement.
type APIOperation interface {
	Run(context.Context) *Result
	GuardConcurrency(context.Context, Entity) (*CategorizedError, error)
	Init(context.Context, OperationRequest) (APIOperation, error)
	GetOperationRequest(context.Context) *OperationRequest
}
