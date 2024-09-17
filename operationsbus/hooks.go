package operationsbus

import (
	"context"
)

type BaseOperationHooksInterface interface {
	BeforeInit(ctx context.Context, req OperationRequest)
	AfterInit(ctx context.Context, op APIOperation, req OperationRequest, err error)

	BeforeGuardConcurrency(ctx context.Context, op APIOperation, entity Entity)
	AfterGuardConcurrency(ctx context.Context, op APIOperation, ce *CategorizedError, err error)

	BeforeRun(ctx context.Context, op APIOperation)
	AfterRun(ctx context.Context, op APIOperation, result Result)
}

var _ BaseOperationHooksInterface = &HookedApiOperation{}

type HookedApiOperation struct {
	Operation      APIOperation
	OperationHooks []BaseOperationHooksInterface
}

func (h *HookedApiOperation) BeforeInit(ctx context.Context, req OperationRequest) {
}
func (h *HookedApiOperation) AfterInit(ctx context.Context, op APIOperation, req OperationRequest, err error) {
}
func (h *HookedApiOperation) BeforeGuardConcurrency(ctx context.Context, op APIOperation, entity Entity) {
}
func (h *HookedApiOperation) AfterGuardConcurrency(ctx context.Context, op APIOperation, ce *CategorizedError, err error) {
}
func (h *HookedApiOperation) BeforeRun(ctx context.Context, op APIOperation) {}
func (h *HookedApiOperation) AfterRun(ctx context.Context, op APIOperation, result Result) {
}

// type HookedApiOperation struct {
//   apiOperation APIOperation
// }

func (h *HookedApiOperation) Init(ctx context.Context, opReq OperationRequest) (APIOperation, error) {
	for _, hook := range h.OperationHooks {
		hook.BeforeInit(ctx, opReq)
	}

	operation, err := h.Operation.Init(ctx, opReq)

	for _, hook := range h.OperationHooks {
		hook.AfterInit(ctx, h.Operation, opReq, err)
	}

	return operation, err
}

func (h *HookedApiOperation) GuardConcurrency(ctx context.Context, entity Entity) (*CategorizedError, error) {
	for _, hook := range h.OperationHooks {
		hook.BeforeGuardConcurrency(ctx, h.Operation, entity)
	}

	ce, err := h.Operation.GuardConcurrency(ctx, entity)

	for _, hook := range h.OperationHooks {
		hook.AfterGuardConcurrency(ctx, h.Operation, ce, err)
	}

	return ce, err
}

func (h *HookedApiOperation) Run(ctx context.Context) *Result {
	for _, hook := range h.OperationHooks {
		hook.BeforeRun(ctx, h.Operation)
	}

	result := h.Operation.Run(ctx)

	for _, hook := range h.OperationHooks {
		hook.AfterRun(ctx, h.Operation, *result)
	}

	return result
}
