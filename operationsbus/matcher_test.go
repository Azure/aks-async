package operationsbus

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type LongRunning struct{}

var _ APIOperation = (*LongRunning)(nil)

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()

	operation := "LongRunning"
	matcher.Register(operation, &LongRunning{})

	result, exists := matcher.Get(operation)
	if !exists {
		t.Fatalf("Operation %s should exist in the matcher, instead got: %t", operation, exists)
	}

	longRunningOp := &LongRunning{}
	longRunningOpType := reflect.TypeOf(longRunningOp).Elem()
	if result != longRunningOpType {
		t.Fatalf("Expected %s. Instead got: %s", longRunningOpType, result)
	}

	// Retrieve an instance of the type associated with the key operation
	instance, err := matcher.CreateInstance(operation)
	if err != nil {
		fmt.Println("Type not found")
		return
	}

	// Check if the created element is of the correct type.
	if reflect.TypeOf(instance).Elem() != longRunningOpType {
		t.Fatalf("The created instance is not of the correct type")
	}
}

// Example implementation of APIOperation for LongRunning
func (lr *LongRunning) Run(ctx context.Context) *Result {
	fmt.Println("Running LongRunning operation")
	return &Result{}
}

func (lr *LongRunning) Guardconcurrency(ctx context.Context, entity Entity) (*CategorizedError, error) {
	fmt.Println("Guarding concurrency in LongRunning operation")
	return &CategorizedError{}, nil
}

func (lr *LongRunning) EntityFetcher(ctx context.Context) (Entity, error) {
	fmt.Println("Fetching entity in LongRunning operation")
	return nil, nil
}

func (lr *LongRunning) Init(ctx context.Context, req OperationRequest) (APIOperation, error) {
	fmt.Println("Initializing LongRunning operation with request")
	return nil, nil
}

func (lr *LongRunning) GetName(ctx context.Context) string {
	return "LongRunning"
}

func (lr *LongRunning) GetOperationRequest(ctx context.Context) *OperationRequest {
	return &OperationRequest{}
}

func (lr *LongRunning) NewContextForOperation(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
