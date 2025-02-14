package matcher

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
)

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()

	operationName := "LongRunning"
	matcher.Register(operationName, &LongRunning{})

	retrieved, exists := matcher.Get(operationName)
	if !exists {
		t.Fatalf("Operation %s should exist in the matcher, instead got: %t", operationName, exists)
	}

	longRunningOp := &LongRunning{}
	longRunningOpType := reflect.TypeOf(longRunningOp).Elem()
	if retrieved != longRunningOpType {
		t.Fatalf("Expected %s. Instead got: %s", longRunningOpType, retrieved)
	}

	// Retrieve an instance of the type associated with the key operation
	instance, err := matcher.CreateOperationInstance(operationName)
	if err != nil {
		fmt.Println("Type not found")
		return
	}

	// Check if the created element is of the correct type.
	if reflect.TypeOf(instance).Elem() != longRunningOpType {
		t.Fatalf("The created instance is not of the correct type")
	}

	ctx := context.Background()
	_, _ = instance.InitOperation(ctx, operation.OperationRequest{})
	_ = instance.Run(ctx)
	if longOp, ok := instance.(*LongRunning); ok {
		if longOp.num != 2 {
			t.Fatalf("Run did not complete successfully: %d", longOp.num)
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunning type.")
	}
}

func TestMatcher_RegisterAndGetEntity(t *testing.T) {
	matcher := NewMatcher()

	entityKey := "TestEntity"
	lastOperationId := "1"
	matcher.RegisterEntity(entityKey, func(latestOperationId string) entity.Entity {
		return &TestEntity{latestOperationId: latestOperationId}
	})

	// Check if the entity creator is registered correctly
	if _, exists := matcher.EntityCreators[entityKey]; !exists {
		t.Fatalf("Entity creator for key %s should exist in the matcher", entityKey)
	}

	// Create an instance of the entity using the registered creator
	entityInstance := matcher.EntityCreators[entityKey]

	var e entity.Entity
	if f, ok := matcher.EntityCreators[entityKey]; ok {
		e = f(lastOperationId)
	} else {
		t.Fatalf("Expected entity instance of type *TestEntity. Instead got: %T", entityInstance)
	}

	if e.(*TestEntity).latestOperationId != "1" {
		t.Fatalf("Expected entity name to be %s. Instead got: %s", lastOperationId, e.(*TestEntity).latestOperationId)
	}
}

func TestMatcher_CreateEntityInstance(t *testing.T) {
	matcher := NewMatcher()

	entityKey := "TestEntity"
	lastOperationId := "1"
	matcher.RegisterEntity(entityKey, func(latestOperationId string) entity.Entity {
		return &TestEntity{latestOperationId: latestOperationId}
	})

	// Create an instance of the entity using the matcher method
	entityInstance, err := matcher.CreateEntityInstance(entityKey, lastOperationId)
	if err != nil {
		t.Fatalf("Expected no error. Instead got: %v", err)
	}
	if testEntityInstance, ok := entityInstance.(*TestEntity); !ok {
		t.Fatalf("Expected entity instance of type *TestEntity. Instead got: %T", entityInstance)
	} else {
		if testEntityInstance.latestOperationId != lastOperationId {
			t.Fatalf("lastestOperationId of entity doesn't match what was used to create the instance: " + lastOperationId)
		}
	}
	if _, ok := entityInstance.(*TestEntity); !ok {
		t.Fatalf("Expected entity instance of type *TestEntity. Instead got: %T", entityInstance)
	}

	if v := entityInstance.GetLatestOperationID(); v != lastOperationId {
		t.Fatalf("Expected latestOperationId of entity to match lastOperationId: " + lastOperationId)
	}
}

func TestMatcher_CreateEntityInstance_NonExistentKey(t *testing.T) {
	matcher := NewMatcher()

	entityKey := "NonExistentEntity"
	_, err := matcher.CreateEntityInstance(entityKey, "1")
	if err == nil {
		t.Fatalf("Should not return function of non-existing entity.")
	}
}

// Example implementatin of entity.
type TestEntity struct {
	latestOperationId string
}

func (e *TestEntity) GetLatestOperationID() string {
	return e.latestOperationId
}

func NewTestEntity(latestOperationId string) *TestEntity {
	return &TestEntity{
		latestOperationId: latestOperationId,
	}
}

// Example implementation of ApiOperation for LongRunning
type LongRunning struct {
	num int
}

var _ operation.ApiOperation = (*LongRunning)(nil)

func (lr *LongRunning) InitOperation(ctx context.Context, req operation.OperationRequest) (operation.ApiOperation, error) {
	fmt.Println("Initializing LongRunning operation with request")
	lr.num = 1
	return nil, nil
}

func (lr *LongRunning) GuardConcurrency(ctx context.Context, entityInstance entity.Entity) *entity.CategorizedError {
	fmt.Println("Guarding concurrency in LongRunning operation")
	return nil
}

func (lr *LongRunning) Run(ctx context.Context) error {
	fmt.Println("Running LongRunning operation")
	lr.num += 1
	return nil
}

func (lr *LongRunning) GetOperationRequest() *operation.OperationRequest {
	return &operation.OperationRequest{}
}
