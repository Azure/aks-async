package operationsbus

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type LongRunning struct {
	num int
}

var _ ApiOperation = (*LongRunning)(nil)

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()

	operation := "LongRunning"
	matcher.Register(operation, &LongRunning{})

	retrieved, exists := matcher.Get(operation)
	if !exists {
		t.Fatalf("Operation %s should exist in the matcher, instead got: %t", operation, exists)
	}

	longRunningOp := &LongRunning{}
	longRunningOpType := reflect.TypeOf(longRunningOp).Elem()
	if retrieved != longRunningOpType {
		t.Fatalf("Expected %s. Instead got: %s", longRunningOpType, retrieved)
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

	ctx := context.Background()
	_, _ = instance.Init(ctx, OperationRequest{})
	_ = instance.Run(ctx)
	if longOp, ok := instance.(*LongRunning); ok {
		if longOp.num != 2 {
			t.Fatalf("Run did not complete successfully: %d", longOp.num)
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunning type.")
	}
}

// Example implementation of ApiOperation for LongRunning
func (lr *LongRunning) Run(ctx context.Context) error {
	fmt.Println("Running LongRunning operation")
	lr.num += 1
	return nil
}

func (lr *LongRunning) GuardConcurrency(ctx context.Context, entity Entity) *CategorizedError {
	fmt.Println("Guarding concurrency in LongRunning operation")
	return &CategorizedError{}
}

func (lr *LongRunning) Init(ctx context.Context, req OperationRequest) (ApiOperation, error) {
	fmt.Println("Initializing LongRunning operation with request")
	lr.num = 1
	return nil, nil
}
