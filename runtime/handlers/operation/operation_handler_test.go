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
	asyncError "github.com/Azure/aks-async/runtime/errors"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
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
		sampleSettler shuttle.MessageSettler
		message       *azservicebus.ReceivedMessage

		operationMatcher     *matcher.Matcher
		operationName        string
		sampleOp             operation.ApiOperation
		mockEntityController *mocks.MockEntityController
		operationHandler     handlerErrors.ErrorHandlerFunc
		marshaller           shuttle.Marshaller
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
		operationMatcher.Register(ctx, operationName, sampleOp)

		operationMatcher.RegisterEntity(ctx, operationName, func(latestOperationId string) (entity.Entity, error) {
			return mocks.NewMockEntity(ctrl), nil
		})

		mockEntityController = mocks.NewMockEntityController(ctrl)

		req := &operation.OperationRequest{
			OperationName:       "SampleOperation",
			ApiVersion:          "v0.0.1",
			OperationId:         "0",
			EntityId:            "1",
			EntityType:          "Cluster",
			RetryCount:          0,
			ExpirationTimestamp: nil,
			Body:                nil,
			HttpMethod:          "",
			Extension:           nil,
		}

		sampleSettler = &settler.SampleMessageSettler{}
		marshaller = &shuttle.DefaultProtoMarshaller{}
		marshalledMessage, err := marshaller.Marshal(req)
		if err != nil {
			return
		}
		message = convert.ConvertToReceivedMessage(marshalledMessage)
		sampleSettler = &settler.SampleMessageSettler{}

		operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController, marshaller)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// TODO(mheberling): Match error types not error messages.
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
			Expect(err.Error()).To(ContainSubstring("Error unmarshalling message"))
		})

		It("should throw an error while creating a hooked instance", func() {
			operationMatcher = matcher.NewMatcher()
			operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController, marshaller)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).To(HaveOccurred())
			Expect(err.Message).To(ContainSubstring("Operation type doesn't exist in the matcher:"))
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
			aerr := &asyncError.AsyncError{OriginalError: errors.New("Random error")}
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, aerr)
			err := operationHandler(ctx, sampleSettler, message)
			Expect(err).ToNot(BeNil())
		})

		It("should throw an error while GuardConcurrency", func() {
			req := &operation.OperationRequest{
				OperationId:   "2",
				OperationName: operationName,
			}
			marshalledOperation, err := marshaller.Marshal(req)
			Expect(err).To(BeNil())
			message.Body = marshalledOperation.Body

			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			ce := operationHandler(ctx, sampleSettler, message)
			Expect(ce).ToNot(BeNil())
		})

		It("should throw an error while Run", func() {
			req := &operation.OperationRequest{
				OperationId:   "3",
				OperationName: operationName,
			}
			marshalledOperation, err := marshaller.Marshal(req)
			Expect(err).To(BeNil())

			message.Body = marshalledOperation.Body
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
