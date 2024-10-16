package operationsbus

import (
	"context"
)

type BaseOperationHooksInterface interface {
	BeforeInitOperation(ctx context.Context, req OperationRequest)
	AfterInitOperation(ctx context.Context, op *ApiOperation, req OperationRequest, err error)

	BeforeGuardConcurrency(ctx context.Context, op *ApiOperation, entity Entity)
	AfterGuardConcurrency(ctx context.Context, op *ApiOperation, ce *CategorizedError)

	BeforeRun(ctx context.Context, op *ApiOperation)
	AfterRun(ctx context.Context, op *ApiOperation, err error)
}

var _ BaseOperationHooksInterface = &HookedApiOperation{}

type HookedApiOperation struct {
	Operation      *ApiOperation
	OperationHooks []BaseOperationHooksInterface
}

func (h *HookedApiOperation) BeforeInitOperation(ctx context.Context, req OperationRequest) {
}
func (h *HookedApiOperation) AfterInitOperation(ctx context.Context, op *ApiOperation, req OperationRequest, err error) {
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op *ApiOperation, entity Entity) {
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op *ApiOperation, ce *CategorizedError) {
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op *ApiOperation) {}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op *ApiOperation, err error) {
}

func (h *HookedApiOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, error) {
	for _, hook := range h.OperationHooks {
		hook.BeforeInitOperation(ctx, opReq)
	}

	operation, err := (*h.Operation).InitOperation(ctx, opReq)

	for _, hook := range h.OperationHooks {
		hook.AfterInitOperation(ctx, h.Operation, opReq, err)
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, entity Entity) *CategorizedError {
	for _, hook := range h.OperationHooks {
		hook.BeforeGuardConcurrency(ctx, h.Operation, entity)
	}

	ce := (*h.Operation).GuardConcurrency(ctx, entity)

	for _, hook := range h.OperationHooks {
		hook.AfterGuardConcurrency(ctx, h.Operation, ce)
	}

	return ce
}

func (h *HookedApiOperation) Run(ctx context.Context) error {
	for _, hook := range h.OperationHooks {
		hook.BeforeRun(ctx, h.Operation)
	}

	err := (*h.Operation).Run(ctx)

	for _, hook := range h.OperationHooks {
		hook.AfterRun(ctx, h.Operation, err)
	}

	return err
}
