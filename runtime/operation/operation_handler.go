package operation

import (
	"context"

	"github.com/Azure/aks-async/runtime/entity"
)

// ApiOperation is the interface all operations will need to implement.
type ApiOperation interface {
	// Initialize the operation with any clients and variables that are required.
	// Only the value of the OperationRequest is passed as an argument
	// (not a pointer to it) since the OperationRequest should not be modified in this
	// function due to receiving the request from the service bus and they can't be modified.
	// In case that the OperationRequest needs to be modified, then a new message with the
	// correct OperationRequest values should be sent via the service bus (or the message
	// broker of choice).
	InitOperation(context.Context, OperationRequest) (ApiOperation, error)
	// GuardConcurrency ensures that this operation is the latest operation that should be
	// running to modify the Entity. If it fails, it should return the CategorizedError.
	GuardConcurrency(context.Context, entity.Entity) *entity.CategorizedError
	// Run will simply run the operation logic required.
	Run(context.Context) error
	// Return the OperationRequest, which may be required by other functions and structs
	// that interact with the operation, such as the EntityController or the operation hooks
	// also ensuring that every operation stored their initial OperationRequest.
	GetOperationRequest() *OperationRequest
}
