package operationsbus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
)

type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req OperationRequest) error
	AfterInitOperation(ctx context.Context, op *ApiOperation, req OperationRequest, err error) error

	BeforeGuardConcurrency(ctx context.Context, op *ApiOperation, entity Entity) error
	AfterGuardConcurrency(ctx context.Context, op *ApiOperation, ce *CategorizedError) error

	BeforeRun(ctx context.Context, op *ApiOperation) error
	AfterRun(ctx context.Context, op *ApiOperation, err error) error
}

// var _ BaseOperationHooksInterface = &HookedApiOperation{}

type HookedApiOperation struct {
	Operation      *ApiOperation
	OperationHooks []BaseOperationHooksInterface
}

func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req OperationRequest) error {
	return nil
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op *ApiOperation, req OperationRequest, err error) error {
	return nil
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op *ApiOperation, entity Entity) error {
	return nil
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op *ApiOperation, ce *CategorizedError) error {
	return nil
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op *ApiOperation) error {
	return nil
}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op *ApiOperation, err error) error {
	return nil
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, error) {
	var herr error
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeInitOperation(ctx, opReq)
		if herr != nil {
			return nil, herr
		}
	}

	logger := ctxlogger.GetLogger(ctx)
	operation, err := (*h.Operation).InitOperation(ctx, opReq)

	for _, hook := range h.OperationHooks {
		logger.Info("Before Init in hook")
		herr = hook.AfterInitOperation(ctx, h.Operation, opReq, err)
		logger.Info("After Init in hook")
		if herr != nil {
			return nil, herr
		}
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, entity Entity) *CategorizedError {
	var herr error
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeGuardConcurrency(ctx, h.Operation, entity)
		if herr != nil {
			return &CategorizedError{
				Message: herr.Error(),
				Err:     herr,
			}
		}
	}

	ce := (*h.Operation).GuardConcurrency(ctx, entity)

	for _, hook := range h.OperationHooks {
		herr = hook.AfterGuardConcurrency(ctx, h.Operation, ce)
		if herr != nil {
			return &CategorizedError{
				Message: herr.Error(),
				Err:     herr,
			}
		}
	}

	return ce
}

func (h *HookedApiOperation) Run(ctx context.Context) error {
	var herr error
	for _, hook := range h.OperationHooks {
		herr = hook.BeforeRun(ctx, h.Operation)
		if herr != nil {
			return herr
		}
	}

	err := (*h.Operation).Run(ctx)

	for _, hook := range h.OperationHooks {
		herr = hook.AfterRun(ctx, h.Operation, err)
		if herr != nil {
			return herr
		}
	}

	return err
}

type OperationControllerHook struct {
	opController OperationController
	BaseOperationHooksInterface
}

func (h *OperationControllerHook) BeforeInitOperation(ctx context.Context, req OperationRequest) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Before BeforeInitOperation")
	err := h.opController.OperationInProgress(ctx, req.OperationId)
	if err != nil {
		return err
	}
	logger.Info("After BeforeInitOperation")
	return nil
}

func (h *OperationControllerHook) AfterInitOperation(ctx context.Context, op *ApiOperation, req OperationRequest, err error) error {
	return nil
}
func (h *OperationControllerHook) BeforeGuardConcurrency(ctx context.Context, op *ApiOperation, entity Entity) error {
	return nil
}
func (h *OperationControllerHook) AfterGuardConcurrency(ctx context.Context, op *ApiOperation, ce *CategorizedError) error {
	return nil
}
func (h *OperationControllerHook) BeforeRun(ctx context.Context, op *ApiOperation) error {
	return nil
}
func (h *OperationControllerHook) AfterRun(ctx context.Context, op *ApiOperation, err error) error {
	if err == nil {
		opreq := (*op).GetOperationRequest()
		err = h.opController.OperationCompleted(ctx, opreq.OperationId)
		if err != nil {
			return err
		}
	}
	return nil
}
