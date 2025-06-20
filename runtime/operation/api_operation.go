package operation

import (
	"context"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/errors"
)

// ApiOperation is the interface all operations will need to implement.
type ApiOperation interface {
	// Initialize the operation with any clients and variables that are required.
	// Can simply return itself after initializing all the required values.
	InitOperation(context.Context, *OperationRequest) (ApiOperation, *errors.AsyncError)
	// GuardConcurrency ensures that this operation is the latest operation that should be
	// running to modify the Entity. If it fails, it should return the AsyncError.
	GuardConcurrency(context.Context, entity.Entity) *errors.AsyncError
	// Run will simply run the operation logic required.
	Run(context.Context) *errors.AsyncError
	// Return the OperationRequest, which may be required by other functions and structs
	// that interact with the operation, such as the EntityController or the operation hooks
	// also ensuring that every operation stored their initial OperationRequest.
	GetOperationRequest() *OperationRequest
}
