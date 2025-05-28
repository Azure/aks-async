package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/Azure/aks-async/mocks"
	"github.com/Azure/aks-async/runtime/entity"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestQoSErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperationHandler Suite")
}

var _ = Describe("OperationHandler", func() {
	var (
		ctrl          *gomock.Controller
		ctx           context.Context
		buf           bytes.Buffer
		operationId   string
		sampleSettler shuttle.MessageSettler
		message       *azservicebus.ReceivedMessage

		operationMatcher     *matcher.Matcher
		operationName        string
		sampleOp             operation.ApiOperation
		mockEntityController *mocks.MockEntityController
		operationHandler     handlerErrors.ErrorHandlerFunc
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		// Need to create an actual operation because if we use mocks the hooks will throw a nil
		// pointer error since it's using new instance created by the matcher which the mock can't
		// reference with EXPECT() calls.
		operationName = "SampleOperation"
		sampleOp = &sampleOperation.SampleOperation{}

		operationMatcher = matcher.NewMatcher()
		operationMatcher.Register(operationName, sampleOp)

		operationMatcher.RegisterEntity(operationName, func(latestOperationId string) entity.Entity {
			return mocks.NewMockEntity(ctrl)
		})

		mockEntityController = mocks.NewMockEntityController(ctrl)

		operationId = "0"
		req := &operation.OperationRequest{
			OperationId:   operationId,
			OperationName: operationName,
		}
		marshalledOperation, err := json.Marshal(req)
		if err != nil {
			return
		}
		message = &azservicebus.ReceivedMessage{
			Body: marshalledOperation,
		}
		sampleSettler = &settler.SampleMessageSettler{}

		operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("mock testing", func() {
		It("should not throw an error", func() {
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).To(BeNil())
		})
		It("should throw an error while unmarshalling", func() {
			invalidMarshalledMessage := &azservicebus.ReceivedMessage{
				Body: []byte(`invalid json`),
			}
			err := operationHandler(ctx, sampleSettler, invalidMarshalledMessage)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&handlerErrors.NonRetryError{Message: "Error unmarshalling message: invalid character 'i' looking for beginning of value"}))
		})
		It("should throw an error while creating a hooked instance", func() {
			operationMatcher = matcher.NewMatcher()
			operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&handlerErrors.NonRetryError{Message: "Error creating operation instance: The ApiOperation doesn't exist in the map: SampleOperation"}))
		})
		It("should throw an error while InitOperation", func() {
			req := &operation.OperationRequest{
				OperationId:   "1",
				OperationName: operationName,
			}
			marshalledOperation, err := json.Marshal(req)
			Expect(err).To(BeNil())

			message.Body = marshalledOperation
			err = operationHandler(ctx, sampleSettler, message)
			Expect(err).ToNot(BeNil())
		})
		It("should throw an error in EntityController", func() {
			randomError := errors.New("Random error")
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, randomError)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).ToNot(BeNil())
		})
		It("should throw an error while GuardConcurrency", func() {
			req := &operation.OperationRequest{
				OperationId:   "2",
				OperationName: operationName,
			}
			marshalledOperation, err := json.Marshal(req)
			Expect(err).To(BeNil())
			message.Body = marshalledOperation

			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			ce := operationHandler(ctx, sampleSettler, message)
			Expect(ce).ToNot(BeNil())
		})
		It("should throw an error while Run", func() {
			req := &operation.OperationRequest{
				OperationId:   "3",
				OperationName: operationName,
			}
			marshalledOperation, err := json.Marshal(req)
			Expect(err).To(BeNil())

			message.Body = marshalledOperation
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			err = operationHandler(ctx, sampleSettler, message)
			Expect(err).ToNot(BeNil())
		})
		It("should throw an error while Settling", func() {
			failureContentType := "failure_test"
			message.ContentType = &failureContentType
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).ToNot(BeNil())
		})
	})
})
