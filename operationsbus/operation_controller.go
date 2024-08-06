package operationsbus

import (
	"context"
)

// Operationcontroller is the interface that handles updating the database with the correct operation state.
type OperationController interface {
	OperationCancel(context.Context, string) error
	OperationInProgress(context.Context, string) error
	OperationTimeout(context.Context, string) error
	OperationCompleted(context.Context, string) error
	OperationPending(context.Context, string) error
	OperationUnknown(context.Context, string) error
	OperationGetEntity(context.Context, OperationRequest) (Entity, error)
}
