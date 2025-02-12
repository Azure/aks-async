package operationsbus

import (
	"context"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
)

// Hooks are used to extend the usability of the operations, and to let the user modify the behavior
// of the different methods we enforce in case they want to change the inputs or outputs.
type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req OperationRequest) error
	AfterInitOperation(ctx context.Context, op ApiOperation, req OperationRequest, err error) error

	BeforeGuardConcurrency(ctx context.Context, op ApiOperation, entity Entity) error
	AfterGuardConcurrency(ctx context.Context, op ApiOperation, ce *CategorizedError) error

	BeforeRun(ctx context.Context, op ApiOperation) error
	AfterRun(ctx context.Context, op ApiOperation, err error) error
}

type HookedApiOperation struct {
	Operation      ApiOperation
	OperationHooks []BaseOperationHooksInterface
}

// HookedApiOperation implements the methods of the BaseOperationHooksInterface to allow the user to
// implement only the hooks they need (e.g. only implement the Before/AfterRun hooks),
// instead of having to implement all of them.
func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req OperationRequest) error {
	return nil
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op ApiOperation, req OperationRequest, err error) error {
	return nil
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op ApiOperation, entity Entity) error {
	return nil
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op ApiOperation, ce *CategorizedError) error {
	return nil
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op ApiOperation) error {
	return nil
}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op ApiOperation, err error) error {
	return nil
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, error) {
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
	operation, err := h.Operation.InitOperation(ctx, opReq)

	logger.Info("Running AfterInit hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterInitOperation(ctx, h.Operation, opReq, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterInit hook: " + herr.Error())
			return nil, herr
		}
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, entity Entity) *CategorizedError {
	logger := ctxlogger.GetLogger(ctx)
	var herr error
	logger.Info("Running BeforeGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeGuardConcurrency(ctx, h.Operation, entity)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeGuardConcurrency hook: " + herr.Error())
			return &CategorizedError{
				Message: herr.Error(),
				Err:     herr,
			}
		}
	}

	logger.Info("Running operation guard concurrency.")
	ce := h.Operation.GuardConcurrency(ctx, entity)

	logger.Info("Running AfterGuardConcurrency hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterGuardConcurrency(ctx, h.Operation, ce)
		if herr != nil {
			logger.Error("Something went wrong running a AfterGuardConcurrency hook: " + herr.Error())
			return &CategorizedError{
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
		herr = hook.BeforeRun(ctx, h.Operation)
		if herr != nil {
			logger.Error("Something went wrong running a BeforeRun hook: " + herr.Error())
			return herr
		}
	}

	logger.Info("Running operation run.")
	err := h.Operation.Run(ctx)

	logger.Info("Running AfterRun hooks.")
	for _, hook := range h.OperationHooks {
		herr = hook.AfterRun(ctx, h.Operation, err)
		if herr != nil {
			logger.Error("Something went wrong running a AfterRun hook: " + herr.Error())
			return herr
		}
	}

	return err
}
