package operationsbus

import (
	"context"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
)

// Hooks are used to extend the usability of the operations, and to let the user modify the behavior
// of the different methods we enforce in case they want to change the inputs or outputs.
type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req OperationRequest) *AsyncError
	AfterInitOperation(ctx context.Context, op ApiOperation, req OperationRequest, asyncError *AsyncError) *AsyncError

	BeforeGuardConcurrency(ctx context.Context, op ApiOperation, entity Entity) *AsyncError
	AfterGuardConcurrency(ctx context.Context, op ApiOperation, asyncError *AsyncError) *AsyncError

	BeforeRun(ctx context.Context, op ApiOperation) *AsyncError
	AfterRun(ctx context.Context, op ApiOperation, asyncError *AsyncError) *AsyncError
}

type HookedApiOperation struct {
	Operation      ApiOperation
	OperationHooks []BaseOperationHooksInterface
}

// HookedApiOperation implements the methods of the BaseOperationHooksInterface to allow the user to
// implement only the hooks they need (e.g. only implement the Before/AfterRun hooks),
// instead of having to implement all of them.
func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req OperationRequest) *AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op ApiOperation, req OperationRequest, err *AsyncError) *AsyncError {
	return nil
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op ApiOperation, entity Entity) *AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op ApiOperation, asyncErr *AsyncError) *AsyncError {
	return nil
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op ApiOperation) *AsyncError {
	return nil
}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op ApiOperation, err *AsyncError) *AsyncError {
	return nil
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, *AsyncError) {
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
	operation, err := h.Operation.InitOperation(ctx, opReq)

	logger.Info("Running AfterInit hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterInitOperation(ctx, h.Operation, opReq, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterInit hook: " + herr.Error())
			return nil, herr
		}
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, entity Entity) *AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running BeforeGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.BeforeGuardConcurrency(ctx, h.Operation, entity)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeGuardConcurrency hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation guard concurrency.")
	asyncError := h.Operation.GuardConcurrency(ctx, entity)

	logger.Info("Running AfterGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterGuardConcurrency(ctx, h.Operation, asyncError)
		if herr != nil {
			logger.Error("Something went wrong running a AfterGuardConcurrency hook: " + herr.Error())
			return herr
		}
	}

	return asyncError
}

func (h *HookedApiOperation) Run(ctx context.Context) *AsyncError {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Running BeforeRun hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.BeforeRun(ctx, h.Operation)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeRun hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation run.")
	err := h.Operation.Run(ctx)

	logger.Info("Running AfterRun hooks.")
	for _, hook := range h.OperationHooks {
		herr := hook.AfterRun(ctx, h.Operation, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterRun hook: " + herr.Error())
			return herr
		}
	}

	return err
}
