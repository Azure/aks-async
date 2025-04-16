package matcher

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Matcher Suite")
}

var _ = Describe("Matcher", func() {
	var (
		matcher           *Matcher
		operationName     string
		sampleOp          *sampleOperation.SampleOperation
		longRunningOpType reflect.Type
		ctx               context.Context
	)

	BeforeEach(func() {
		matcher = NewMatcher()
		operationName = "LongRunning"
		sampleOp = &sampleOperation.SampleOperation{}
		longRunningOpType = reflect.TypeOf(sampleOp).Elem()
		ctx = context.Background()
	})

	Describe("Register and Get Operation", func() {
		It("should register and retrieve the operation type", func() {
			matcher.Register(operationName, sampleOp)

			retrieved, exists := matcher.Get(operationName)
			Expect(exists).To(BeTrue(), fmt.Sprintf("Operation %s should exist in the matcher", operationName))
			Expect(retrieved).To(Equal(longRunningOpType), fmt.Sprintf("Expected %s. Instead got: %s", longRunningOpType, retrieved))
		})
	})

	Describe("Create Operation Instance", func() {
		It("should create an instance of the registered operation type", func() {
			matcher.Register(operationName, sampleOp)

			instance, err := matcher.CreateOperationInstance(operationName)
			Expect(err).NotTo(HaveOccurred(), "Type not found")
			Expect(reflect.TypeOf(instance).Elem()).To(Equal(longRunningOpType), "The created instance is not of the correct type")

			_, _ = instance.InitOperation(ctx, operation.OperationRequest{})
			err = instance.Run(ctx)
			Expect(err).To(BeNil())
			if op, ok := instance.(*sampleOperation.SampleOperation); ok {
				Expect(op.Num).To(Equal(1), "Run did not complete successfully")
			} else {
				Fail("Something went wrong casting the operation to LongRunning type.")
			}
		})
	})

	Describe("Register and Get Entity", func() {
		It("should register and retrieve the entity creator", func() {
			entityKey := "TestEntity"
			lastOperationId := "1"
			matcher.RegisterEntity(entityKey, func(latestOperationId string) entity.Entity {
				return &TestEntity{latestOperationId: latestOperationId}
			})

			Expect(matcher.EntityCreators).To(HaveKey(entityKey), fmt.Sprintf("Entity creator for key %s should exist in the matcher", entityKey))

			entityInstance := matcher.EntityCreators[entityKey]
			var e entity.Entity
			if f, ok := matcher.EntityCreators[entityKey]; ok {
				e = f(lastOperationId)
			} else {
				Fail(fmt.Sprintf("Expected entity instance of type *TestEntity. Instead got: %T", entityInstance))
			}

			Expect(e.(*TestEntity).latestOperationId).To(Equal(lastOperationId), fmt.Sprintf("Expected entity name to be %s. Instead got: %s", lastOperationId, e.(*TestEntity).latestOperationId))
		})
	})

	Describe("Create Entity Instance", func() {
		It("should create an instance of the registered entity type", func() {
			entityKey := "TestEntity"
			lastOperationId := "1"
			matcher.RegisterEntity(entityKey, func(latestOperationId string) entity.Entity {
				return &TestEntity{latestOperationId: latestOperationId}
			})

			entityInstance, err := matcher.CreateEntityInstance(entityKey, lastOperationId)
			Expect(err).NotTo(HaveOccurred(), "Expected no error")
			Expect(entityInstance).To(BeAssignableToTypeOf(&TestEntity{}), fmt.Sprintf("Expected entity instance of type *TestEntity. Instead got: %T", entityInstance))
			Expect(entityInstance.(*TestEntity).latestOperationId).To(Equal(lastOperationId), "lastestOperationId of entity doesn't match what was used to create the instance")

			Expect(entityInstance.GetLatestOperationID()).To(Equal(lastOperationId), "Expected latestOperationId of entity to match lastOperationId")
		})

		It("should return an error for non-existent entity key", func() {
			entityKey := "NonExistentEntity"
			_, err := matcher.CreateEntityInstance(entityKey, "1")
			Expect(err).To(HaveOccurred(), "Should not return function of non-existing entity.")
		})
	})
})

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
