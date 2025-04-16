package operation

import (
	"context"
	"errors"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
)

// Need to create an actual operation because if we use mocks the hooks will throw a nil
// pointer error since it's using new instance created by the matcher which the mock can't
// reference with EXPECT() calls.
// Sample operation
var _ operation.ApiOperation = &SampleOperation{}

type SampleOperation struct {
	opReq operation.OperationRequest
	Num   int
}

func (l *SampleOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, error) {
	if opReq.OperationId == "1" {
		return nil, errors.New("No OperationId")
	}
	l.opReq = opReq
	l.Num = 1
	return nil, nil
}

func (l *SampleOperation) GuardConcurrency(ctx context.Context, entityInstance entity.Entity) *entity.CategorizedError {
	if l.opReq.OperationId == "2" {
		ce := &entity.CategorizedError{Err: errors.New("Incorrect OperationId")}
		return ce
	}
	return nil
}

func (l *SampleOperation) Run(ctx context.Context) error {
	if l.opReq.OperationId == "3" {
		return errors.New("Incorrect OperationId")
	}
	return nil
}

func (l *SampleOperation) GetOperationRequest() *operation.OperationRequest {
	return &l.opReq
}
