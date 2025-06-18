package operation

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/operation"
	sampleErrorHandler "github.com/Azure/aks-async/runtime/testutils/error_handler"
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
	RunSpecs(t, "OperationContainerHandler Suite")
}

var _ = Describe("OperationContainerHandler", func() {
	var (
		ctrl                         *gomock.Controller
		ctx                          context.Context
		buf                          bytes.Buffer
		operationId                  string
		sampleSettler                shuttle.MessageSettler
		message                      *azservicebus.ReceivedMessage
		operationContainerClient     *ocMock.MockOperationContainerClient
		marshaller                   shuttle.Marshaller
		updateOperationStatusRequest *oc.UpdateOperationStatusRequest
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		sampleSettler = &settler.SampleMessageSettler{}
		operationId = "0"
		req := &operation.OperationRequest{
			OperationName:       "SampleOperation",
			ApiVersion:          "v0.0.1",
			OperationId:         operationId,
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
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("mock testing", func() {
		var (
			operationContainerHandler handlerErrors.ErrorHandlerFunc
		)

		BeforeEach(func() {
			// Need to reset the UpdateOperationStatusRequest since the status request sent is different for each test.
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: operationId,
			}
		})

		Context("normal flow", func() {
			It("should not throw an error", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)

				updateOperationStatusRequest.Status = oc.Status_SUCCEEDED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
				err := operationContainerHandler(ctx, sampleSettler, message)
				Expect(err).To(BeNil())
			})

			It("should handle operationContainer client returning an error", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)

				updateOperationStatusRequest.Status = oc.Status_SUCCEEDED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

				returnedErr := errors.New("Random error")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
				err := operationContainerHandler(ctx, sampleSettler, message)
				Expect(err).ToNot(BeNil())
				Expect(errors.Is(err, returnedErr)).To(BeTrue())
			})
		})

		Context("Errors", func() {
			Context("NonRetryError", func() {
				It("should handle a NonRetryError", func() {
					nonRetryError := &asyncErrors.NonRetryError{
						Message: "NonRetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nonRetryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_FAILED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(errors.Is(err, nonRetryError)).To(BeTrue())
				})

				It("should handle client error while handling a NonRetryError", func() {
					nonRetryError := &asyncErrors.NonRetryError{
						Message: "NonRetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nonRetryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("Random error")
					updateOperationStatusRequest.Status = oc.Status_FAILED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(errors.Is(err, nonRetryError)).To(BeTrue())
				})
			})

			Context("RetryError", func() {
				It("should handle a RetryError", func() {
					retryError := &asyncErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(retryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(errors.Is(err, retryError)).To(BeTrue())
				})

				It("should handle client error while handling a RetryError", func() {
					retryError := &asyncErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(retryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("Random error")
					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)

					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(errors.Is(err, retryError)).To(BeTrue())
				})
			})

			Context("default", func() {
				It("should handle a default", func() {
					defaultError := errors.New("default error")

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(defaultError), operationContainerClient, marshaller)

					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
				})
			})
		})
	})
})
