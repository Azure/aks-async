package operationsbus

import (
	"context"
)

type EntityController interface {
	GetEntity(context.Context, OperationRequest) (Entity, error)
}
