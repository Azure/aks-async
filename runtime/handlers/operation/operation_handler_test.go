package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/Azure/aks-async/mocks"
	"github.com/Azure/aks-async/runtime/entity"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("OperationContainerHandler Testing", func() {
	var (
		ctrl        *gomock.Controller
		ctx         context.Context
		buf         bytes.Buffer
		operationId string
		settler     shuttle.MessageSettler
		message     *azservicebus.ReceivedMessage

		operationMatcher     *matcher.Matcher
		operationName        string
		sampleOperation      operation.ApiOperation
		mockEntityController *mocks.MockEntityController
		operationHandler     handlerErrors.ErrorHandlerFunc
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		operationName = "SampleOperation"
		sampleOperation = &SampleOperation{}

		operationMatcher = matcher.NewMatcher()
		operationMatcher.Register(operationName, sampleOperation)

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
		settler = &fakeMessageSettler{}

		operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("mock testing", func() {
		It("should not throw an error", func() {

			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			err := operationHandler(ctx, settler, message)
			Expect(err).To(BeNil())
		})
		It("should throw an error while unmarshalling", func() {
			invalidMarshalledMessage := &azservicebus.ReceivedMessage{
				Body: []byte(`invalid json`),
			}
			err := operationHandler(ctx, settler, invalidMarshalledMessage)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&handlerErrors.NonRetryError{Message: "Error unmarshalling message."}))
		})
		It("should throw an error while creating a hooked instance", func() {
			operationMatcher = matcher.NewMatcher()
			operationHandler = NewOperationHandler(operationMatcher, nil, mockEntityController)
			err := operationHandler(ctx, settler, message)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&handlerErrors.NonRetryError{Message: "Error creating operation instance."}))
		})
		It("should throw an error while InitOperation", func() {
			req := &operation.OperationRequest{
				OperationId:   "1",
				OperationName: operationName,
			}
			marshalledOperation, err := json.Marshal(req)
			Expect(err).To(BeNil())

			message.Body = marshalledOperation
			err = operationHandler(ctx, settler, message)
			Expect(err).ToNot(BeNil())
		})
		It("should throw an error in EntityController", func() {
			randomError := errors.New("Random error")
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, randomError)
			err := operationHandler(ctx, settler, message)
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
			ce := operationHandler(ctx, settler, message)
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
			err = operationHandler(ctx, settler, message)
			Expect(err).ToNot(BeNil())
		})
		It("should throw an error while Settling", func() {
			failureContentType := "failure_test"
			message.ContentType = &failureContentType
			mockEntityController.EXPECT().GetEntity(gomock.Any(), gomock.Any()).Return(nil, nil)
			err := operationHandler(ctx, settler, message)
			Expect(err).ToNot(BeNil())
		})
	})
})

// Need to create an actual operation because if we use mocks the hooks will throw a nil
// pointer error since it's using new instance created by the matcher which the mock can't
// reference with EXPECT() calls.
// Sample operation
var _ operation.ApiOperation = &SampleOperation{}

type SampleOperation struct {
	opReq operation.OperationRequest
	num   int
}

func (l *SampleOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, error) {
	if opReq.OperationId == "1" {
		return nil, errors.New("No OperationId")
	}
	l.opReq = opReq
	l.num = 1
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
