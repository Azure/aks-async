package hooks

import (
	"context"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
)

// Hooks are used to extend the usability of the operations, and to let the user modify the behavior
// of the different methods we enforce in case they want to change the inputs or outputs.
type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req operation.OperationRequest) *errors.AsyncError
	AfterInitOperation(ctx context.Context, op operation.ApiOperation, req operation.OperationRequest, asyncError *errors.AsyncError) *errors.AsyncError

	BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, e entity.Entity) *errors.AsyncError
	AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, asyncError *errors.AsyncError) *errors.AsyncError

	BeforeRun(ctx context.Context, op operation.ApiOperation) *errors.AsyncError
	AfterRun(ctx context.Context, op operation.ApiOperation, asyncError *errors.AsyncError) *errors.AsyncError
}

type HookedApiOperation struct {
	OperationInstance operation.ApiOperation
	OperationHooks    []BaseOperationHooksInterface
}

// HookedApiOperation implements the methods of the BaseOperationHooksInterface to allow the user to
// implement only the hooks they need (e.g. only implement the Before/AfterRun hooks),
// instead of having to implement all of them.
func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req operation.OperationRequest) *errors.AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op operation.ApiOperation, req operation.OperationRequest, err *errors.AsyncError) *errors.AsyncError {
	return nil
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op operation.ApiOperation, e entity.Entity) *errors.AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op operation.ApiOperation, asyncErr *errors.AsyncError) *errors.AsyncError {
	return nil
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op operation.ApiOperation) *errors.AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op operation.ApiOperation, err *errors.AsyncError) *errors.AsyncError {
	return nil
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, *errors.AsyncError) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running BeforeInit hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.BeforeInitOperation(ctx, opReq)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeInit hook: " + herr.Error())
			return nil, herr
		}
	}

	logger.Info("Running operation init.")
	operation, err := h.OperationInstance.InitOperation(ctx, opReq)

	logger.Info("Running AfterInit hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterInitOperation(ctx, h.OperationInstance, opReq, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterInit hook: " + herr.Error())
			return nil, herr
		}
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, e entity.Entity) *errors.AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running BeforeGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.BeforeGuardConcurrency(ctx, h.OperationInstance, e)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeGuardConcurrency hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation guard concurrency.")
	asyncError := h.OperationInstance.GuardConcurrency(ctx, e)

	logger.Info("Running AfterGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterGuardConcurrency(ctx, h.OperationInstance, asyncError)
		if herr != nil {
			logger.Error("Something went wrong running a AfterGuardConcurrency hook: " + herr.Error())
			return herr
		}
	}

	return asyncError
}

func (h *HookedApiOperation) Run(ctx context.Context) *errors.AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running BeforeRun hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.BeforeRun(ctx, h.OperationInstance)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeRun hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation run.")
	err := h.OperationInstance.Run(ctx)

	logger.Info("Running AfterRun hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterRun(ctx, h.OperationInstance, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterRun hook: " + herr.Error())
			return herr
		}
	}

	return err
}
