package entity

import (
	"context"
	"database/sql"
	"fmt"

	oc "github.com/Azure/OperationContainer/api/v1"
	database "github.com/Azure/aks-async/database"
	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/operation"
)

var _ hooks.BaseOperationHooksInterface = &OperationStatusHook{}

// OperationContainer is already taken care of as a handler because we passed it in to the processor. We only need to update the entity table here.
type OperationStatusHook struct {
	dbClient        *sql.DB
	EntityTableName string
}

func (h *OperationStatusHook) BeforeInitOperation(ctx context.Context, req operation.OperationRequest) error {
	// set operation as in in progress
	inProgressOperationStatus := oc.Status_IN_PROGRESS.String()
	err := h.updateEntityDatabase(ctx, inProgressOperationStatus, req.EntityId, req.EntityType)
	if err != nil {
		return fmt.Errorf("Error updating the entity database of entity with id: %s and type: %s to IN_PROGRESS status: %s", req.EntityId, req.EntityType, err)
	}

	return nil
}

func (h *OperationStatusHook) AfterInitOperation(ctx context.Context, op operation.ApiOperation, req operation.OperationRequest, err error) error {
	// on error set as pending
	if err != nil {
		pendingOperationStatus := oc.Status_PENDING.String()
		err := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType)
		if err != nil {
			return fmt.Errorf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, err)
		}
	}
	return nil
}

func (h *OperationStatusHook) BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, e entity.Entity) error {
	// If there was an error with any function before getting here, it would've been caught. Nothing to do here.
	return nil
}

// TODO(mheberling): define what we want to do with the categorized error. Potentially delete it in favor of a regular error.
func (h *OperationStatusHook) AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, ce *entity.CategorizedError) error {
	// on error set as pending
	if ce != nil {
		req := op.GetOperationRequest()
		pendingOperationStatus := oc.Status_PENDING.String()
		err := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType)
		if err != nil {
			return fmt.Errorf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, err)
		}
	}
	return nil
}

func (h *OperationStatusHook) BeforeRun(ctx context.Context, op operation.ApiOperation) error {
	// If there was an error with any function before getting here, it would've been caught. Nothing to do here.
	return nil
}

func (h *OperationStatusHook) AfterRun(ctx context.Context, op operation.ApiOperation, err error) error {
	req := op.GetOperationRequest()
	if err != nil {
		// on error set as pending
		pendingOperationStatus := oc.Status_PENDING.String()
		err := h.updateEntityDatabase(ctx, pendingOperationStatus, req.EntityId, req.EntityType)
		if err != nil {
			return fmt.Errorf("Error updating the entity database of entity with id: %s and type: %s to PENDING status: %s", req.EntityId, req.EntityType, err)
		}
	} else {
		// no nil error set as complete
		inProgressOperationStatus := oc.Status_COMPLETED.String()
		err := h.updateEntityDatabase(ctx, inProgressOperationStatus, req.EntityId, req.EntityType)
		if err != nil {
			return fmt.Errorf("Error updating the entity database of entity with id: %s and type: %s to COMPLETED status: %s", req.EntityId, req.EntityType, err)
		}
	}
	return nil
}

func (h *OperationStatusHook) updateEntityDatabase(ctx context.Context, newStatus string, entityId string, entityType string) error {
	query := fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND EXISTS (SELECT 1 FROM %s WHERE entity_id = @p2 AND entity_type = @p3)`, h.EntityTableName, h.EntityTableName)
	_, err := database.ExecDb(ctx, h.dbClient, query, newStatus, entityId, entityType)
	if err != nil {
		return fmt.Errorf("Error in operations query: %w", err)
	}
	return nil
}
