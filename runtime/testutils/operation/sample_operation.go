package operation

import (
	"context"
	"errors"

	"github.com/Azure/aks-async/runtime/entity"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/operation"
)

var _ operation.ApiOperation = &SampleOperation{}

type SampleOperation struct {
	opReq *operation.OperationRequest
	Num   int
}

func (l *SampleOperation) InitOperation(ctx context.Context, opReq *operation.OperationRequest) (operation.ApiOperation, *asyncErrors.AsyncError) {
	if opReq.OperationId == "1" {
		return nil, &asyncErrors.AsyncError{OriginalError: errors.New("No OperationId")}
	}
	l.opReq = opReq
	l.Num = 1
	return nil, nil
}

func (l *SampleOperation) GuardConcurrency(ctx context.Context, entityInstance entity.Entity) *asyncErrors.AsyncError {
	if l.opReq.OperationId == "2" {
		err := &asyncErrors.AsyncError{OriginalError: errors.New("Incorrect OperationId")}
		return err
	}
	return nil
}

func (l *SampleOperation) Run(ctx context.Context) *asyncErrors.AsyncError {
	if l.opReq.OperationId == "3" {
		return &asyncErrors.AsyncError{OriginalError: errors.New("Incorrect OperationId")}
	}
	return nil
}

func (l *SampleOperation) GetOperationRequest() *operation.OperationRequest {
	return l.opReq
}
