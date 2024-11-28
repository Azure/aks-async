package operationsbus

import (
	"context"
)

// ApiOperation is the interface all operations will need to implement.
type ApiOperation interface {
	InitOperation(context.Context, OperationRequest) (ApiOperation, error)
	GuardConcurrency(context.Context, Entity) *CategorizedError
	Run(context.Context) error
	GetOperationRequest() *OperationRequest
}
