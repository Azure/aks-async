package hooks

import (
	"context"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
)

// Hooks are used to extend the usability of the operations, and to let the user modify the behavior
// of the different methods we enforce in case they want to change the inputs or outputs.
type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req operation.OperationRequest) error
	AfterInitOperation(ctx context.Context, op operation.ApiOperation, req operation.OperationRequest, err error) error

	BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, operationEntity entity.Entity) error
	AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, ce *entity.CategorizedError) error

	BeforeRun(ctx context.Context, op operation.ApiOperation) error
	AfterRun(ctx context.Context, op operation.ApiOperation, err error) error
}

type HookedApiOperation struct {
	OperationInstance operation.ApiOperation
	OperationHooks    []BaseOperationHooksInterface
}

// HookedApiOperation implements the methods of the BaseOperationHooksInterface to allow the user to
// implement only the hooks they need (e.g. only implement the Before/AfterRun hooks),
// instead of having to implement all of them.
func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req operation.OperationRequest) error {
	return nil
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op operation.ApiOperation, req operation.OperationRequest, err error) error {
	return nil
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, operationEntity entity.Entity) error {
	return nil
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, ce *entity.CategorizedError) error {
	return nil
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op operation.ApiOperation) error {
	return nil
}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op operation.ApiOperation, err error) error {
	return nil
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, error) {
	logger := ctxlogger.GetLogger(ctx)
	var herr error
	logger.Info("Running BeforeInit hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeInitOperation(ctx, opReq)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeInit hook: " + herr.Error())
			return nil, herr
		}
	}

	logger.Info("Running operation init.")
	operation, err := h.OperationInstance.InitOperation(ctx, opReq)

	logger.Info("Running AfterInit hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterInitOperation(ctx, h.OperationInstance, opReq, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterInit hook: " + herr.Error())
			return nil, herr
		}
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, operationEntity entity.Entity) *entity.CategorizedError {
	logger := ctxlogger.GetLogger(ctx)
	var herr error
	logger.Info("Running BeforeGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeGuardConcurrency(ctx, h.OperationInstance, operationEntity)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeGuardConcurrency hook: " + herr.Error())
			return &entity.CategorizedError{
				Message: herr.Error(),
				Err:     herr,
			}
		}
	}

	logger.Info("Running operation guard concurrency.")
	ce := h.OperationInstance.GuardConcurrency(ctx, operationEntity)

	logger.Info("Running AfterGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterGuardConcurrency(ctx, h.OperationInstance, ce)
		if herr != nil {
			logger.Error("Something went wrong running a AfterGuardConcurrency hook: " + herr.Error())
			return &entity.CategorizedError{
				Message: herr.Error(),
				Err:     herr,
			}
		}
	}

	return ce
}

func (h *HookedApiOperation) Run(ctx context.Context) error {
	logger := ctxlogger.GetLogger(ctx)
	var herr error
	logger.Info("Running BeforeRun hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeRun(ctx, h.OperationInstance)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeRun hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation run.")
	err := h.OperationInstance.Run(ctx)

	logger.Info("Running AfterRun hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterRun(ctx, h.OperationInstance, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterRun hook: " + herr.Error())
			return herr
		}
	}

	return err
}
