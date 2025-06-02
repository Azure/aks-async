package operationsbus

import (
	"context"
)

// ApiOperation is the interface all operations will need to implement.
type ApiOperation interface {
	InitOperation(context.Context, OperationRequest) (ApiOperation, *AsyncError)
	GuardConcurrency(context.Context, Entity) *AsyncError
	Run(context.Context) *AsyncError
	GetOperationRequest() *OperationRequest
}
