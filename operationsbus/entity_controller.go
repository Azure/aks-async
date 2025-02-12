package operationsbus

import (
	"context"
)

// EntityController is the interface used to grab the entity from where it's stored, typically a database.
// Using this interface we enforce people to not only implement the GuardConcurrency method in each operations
// but also force them to grab the entity from somewhere, thus avoiding them simply returning always returning
// nil error from GuardConcurrency method.
type EntityController interface {
	GetEntity(context.Context, OperationRequest) (Entity, *AsyncError)
}
