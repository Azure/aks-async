package operationsbus

import (
	"context"
)

// ApiOperation is the interface all operations will need to implement.
type ApiOperation interface {
	Init(context.Context, OperationRequest) (ApiOperation, error)
	GuardConcurrency(context.Context, Entity) *CategorizedError
	Run(context.Context) error
}
