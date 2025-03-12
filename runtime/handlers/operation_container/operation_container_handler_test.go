package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestQoSErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperationContainerHandler Suite")
}

var _ = Describe("OperationContainerHandler", func() {
	var (
		ctrl                     *gomock.Controller
		ctx                      context.Context
		buf                      bytes.Buffer
		operationId              string
		settler                  shuttle.MessageSettler
		message                  *azservicebus.ReceivedMessage
		operationContainerClient *ocMock.MockOperationContainerClient
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		settler = &fakeMessageSettler{}
		operationId = "1"
		req := &operation.OperationRequest{
			OperationId: operationId,
		}
		marshalledOperation, err := json.Marshal(req)
		if err != nil {
			return
		}
		message = &azservicebus.ReceivedMessage{
			Body: marshalledOperation,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("mock testing", func() {
		var (
			operationContainerHandler    handlerErrors.ErrorHandlerFunc
			updateOperationStatusRequest *oc.UpdateOperationStatusRequest
		)
		BeforeEach(func() {
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: operationId,
			}
		})
		Context("no error", func() {
			It("should not throw an error", func() {
				operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nil), operationContainerClient)

				updateOperationStatusRequest.Status = oc.Status_COMPLETED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
				err := operationContainerHandler(ctx, settler, message)
				Expect(err).To(BeNil())
			})
			It("should handle error", func() {
				operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nil), operationContainerClient)

				updateOperationStatusRequest.Status = oc.Status_COMPLETED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

				returnedErr := errors.New("returned error")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
				err := operationContainerHandler(ctx, settler, message)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(returnedErr.Error()))
			})
		})
		Context("retries", func() {
			It("should retry 5 times and fail", func() {
				operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nil), operationContainerClient)
				updateOperationStatusRequest.Status = oc.Status_IN_PROGRESS

				err := errors.New("Some database error.")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				err = operationContainerHandler(ctx, settler, message)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(buf.String(), "Trying again")).To(Equal(5))
			})
			It("should retry 3 times and succeed", func() {
				operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nil), operationContainerClient)
				updateOperationStatusRequest.Status = oc.Status_IN_PROGRESS

				err := errors.New("Some database error.")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err) // Retry 1
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err) // Retry 2
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err) // Retry 3
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil) // Retry 4 succeeds

				completedUpdateOperationStatusRequest := &oc.UpdateOperationStatusRequest{
					OperationId: operationId,
					Status:      oc.Status_COMPLETED,
				}
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, completedUpdateOperationStatusRequest).Return(nil, nil) // Final update to complete
				err = operationContainerHandler(ctx, settler, message)
				Expect(err).To(BeNil())
				Expect(strings.Count(buf.String(), "Trying again")).To(Equal(3))
			})
		})

		Context("Errors", func() {
			Context("NonRetryError", func() {
				It("should handle a NonRetryError", func() {
					nonRetryError := &handlerErrors.NonRetryError{
						Message: "NonRetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nonRetryError), operationContainerClient)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_CANCELLED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, settler, message)
					Expect(err).ToNot(BeNil())
				})
				It("should handle error in update while handling a NonRetryError", func() {
					nonRetryError := &handlerErrors.NonRetryError{
						Message: "NonRetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(nonRetryError), operationContainerClient)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("returned error")
					updateOperationStatusRequest.Status = oc.Status_CANCELLED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
					err := operationContainerHandler(ctx, settler, message)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(Equal(returnedErr.Error()))
				})
			})
			Context("NonRetryError", func() {
				It("should handle a RetryError", func() {
					retryError := &handlerErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(retryError), operationContainerClient)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, settler, message)
					Expect(err).ToNot(BeNil())
				})
				It("should handle an error while handling a RetryError", func() {
					retryError := &handlerErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(retryError), operationContainerClient)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("returned error")
					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
					err := operationContainerHandler(ctx, settler, message)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(Equal(returnedErr.Error()))
				})
			})
			Context("default", func() {
				It("should handle a default", func() {
					defaultError := errors.New("default error")

					operationContainerHandler = NewOperationContainerHandler(SampleErrorHandler(defaultError), operationContainerClient)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					err := operationContainerHandler(ctx, settler, message)
					Expect(err).ToNot(BeNil())
				})
			})
		})
	})
})

func SampleHandler() shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
	}
}

func SampleErrorHandler(testErrorMessage error) handlerErrors.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		return testErrorMessage
	}
}

type fakeMessageSettler struct{}

func (f *fakeMessageSettler) AbandonMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.AbandonMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) CompleteMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *fakeMessageSettler) DeadLetterMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) error {
	return nil
}
func (f *fakeMessageSettler) DeferMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeferMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) RenewMessageLock(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.RenewMessageLockOptions) error {
	return nil
}
