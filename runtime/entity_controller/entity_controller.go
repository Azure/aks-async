package entity_controller

import (
	"context"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
)

// EntityController is the interface used to grab the entity from where it's stored, typically a database.
// Using this interface we enforce people to not only implement the GuardConcurrency method in each operations
// but also force them to grab the entity from somewhere, thus avoiding them always returning a nil error
// from GuardConcurrency method.
type EntityController interface {
	GetEntity(context.Context, operation.OperationRequest) (entity.Entity, error)
}
