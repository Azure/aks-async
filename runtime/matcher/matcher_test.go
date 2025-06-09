package matcher

import (
	"context"
	"errors"
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
		matcher             *Matcher
		operationName       string
		sampleOp            *sampleOperation.SampleOperation
		sampleOperationType reflect.Type
		ctx                 context.Context
	)

	BeforeEach(func() {
		matcher = NewMatcher()
		operationName = "LongRunning"
		sampleOp = &sampleOperation.SampleOperation{}
		sampleOperationType = reflect.TypeOf(sampleOp).Elem()
		ctx = context.Background()
	})

	Describe("Register and Get Operation", func() {
		It("should register and retrieve the operation type", func() {
			matcher.Register(ctx, operationName, sampleOp)
			Expect(matcher.Types).To(HaveKey(operationName))

			retrieved, exists := matcher.Get(ctx, operationName)
			Expect(exists).To(BeTrue())
			Expect(retrieved).To(Equal(sampleOperationType))
		})

		It("should not find unregistered operation type", func() {
			unregisteredOperation := "UnregisteredOperation"
			Expect(matcher.Types).NotTo(HaveKey(unregisteredOperation))

			operation, exists := matcher.Get(ctx, unregisteredOperation)
			Expect(exists).To(BeFalse())
			Expect(operation).To(BeNil())
		})
	})

	Describe("Create Operation Instance", func() {
		It("should create an instance of the registered operation type", func() {
			matcher.Register(ctx, operationName, sampleOp)

			Expect(matcher.Types).To(HaveKey(operationName))
			instance, err := matcher.CreateOperationInstance(ctx, operationName)
			Expect(err).NotTo(HaveOccurred())
			Expect(reflect.TypeOf(instance).Elem()).To(Equal(sampleOperationType))

			_, _ = instance.InitOperation(ctx, &operation.OperationRequest{})
			err = instance.Run(ctx)
			Expect(err).To(BeNil())
			op, ok := instance.(*sampleOperation.SampleOperation)
			Expect(ok).To(BeTrue())
			Expect(op.Num).To(Equal(1))
		})

		It("should fail is key doesn't exist", func() {
			Expect(matcher.Types).NotTo(HaveKey(operationName))
			instance, err := matcher.CreateOperationInstance(ctx, operationName)

			var opErr *OperationKeyLookupError
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &opErr)).To(BeTrue())
			Expect(instance).To(BeNil())
		})
	})

	Describe("Register and Create Entity", func() {
		It("should register and retrieve the entity creator", func() {
			entityKey := "TestEntity"
			lastOperationId := "1"
			matcher.RegisterEntity(ctx, entityKey, func(latestOperationId string) (entity.Entity, error) {
				return &TestEntity{latestOperationId: latestOperationId}, nil
			})

			Expect(matcher.EntityCreators).To(HaveKey(entityKey))

			entityInstance, err := matcher.CreateEntityInstance(ctx, entityKey, lastOperationId)
			Expect(err).To(BeNil())
			Expect(entityInstance).ToNot(BeNil())
			Expect(entityInstance.GetLatestOperationID()).To(Equal(lastOperationId))
		})

		It("should fail if no lastOperationId provided", func() {
			entityKey := "TestEntity"
			Expect(matcher.EntityCreators).NotTo(HaveKey(entityKey))

			entityInstance, err := matcher.CreateEntityInstance(ctx, entityKey, "")
			Expect(entityInstance).To(BeNil())
			Expect(err).To(HaveOccurred())
			var entityErr *EmptyOperationId
			Expect(errors.As(err, &entityErr)).To(BeTrue())
		})

		It("should fail if key doesn't exist in map", func() {
			entityKey := "TestEntity"
			lastOperationId := "1"

			Expect(matcher.EntityCreators).NotTo(HaveKey(entityKey))

			entityInstance, err := matcher.CreateEntityInstance(ctx, entityKey, lastOperationId)
			Expect(entityInstance).To(BeNil())
			Expect(err).To(HaveOccurred())
			var entityErr *EntityCreationKeyLookupError
			Expect(errors.As(err, &entityErr)).To(BeTrue())
		})

		It("should fail if entity creation returns an error", func() {
			entityKey := "TestEntity"
			lastOperationId := "1"
			matcher.RegisterEntity(ctx, entityKey, func(latestOperationId string) (entity.Entity, error) {
				return nil, errors.New("Some error")
			})

			Expect(matcher.EntityCreators).To(HaveKey(entityKey))

			entityInstance, err := matcher.CreateEntityInstance(ctx, entityKey, lastOperationId)
			Expect(entityInstance).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Create Hooked Instance", func() {
		It("should create a hooked instance of the registered operation type", func() {
			matcher.Register(ctx, operationName, sampleOp)
			Expect(matcher.Types).To(HaveKey(operationName))

			hOp, err := matcher.CreateHookedInstance(ctx, operationName, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(hOp).ToNot(BeNil())
		})

		It("should fail if operation is not registered", func() {
			Expect(matcher.Types).ToNot(HaveKey(operationName))

			hOp, err := matcher.CreateHookedInstance(ctx, operationName, nil)
			Expect(hOp).To(BeNil())
			Expect(err).To(HaveOccurred())
			var opKeyErr *OperationKeyLookupError
			Expect(errors.As(err, &opKeyErr)).To(BeTrue())
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
