package operation

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/operation"
	sampleErrorHandler "github.com/Azure/aks-async/runtime/testutils/error_handler"
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
	RunSpecs(t, "OperationContainerHandler Suite")
}

var _ = Describe("OperationContainerHandler", func() {
	var (
		ctrl                     *gomock.Controller
		ctx                      context.Context
		buf                      bytes.Buffer
		operationId              string
		sampleSettler            shuttle.MessageSettler
		message                  *azservicebus.ReceivedMessage
		operationContainerClient *ocMock.MockOperationContainerClient
		marshaller               shuttle.Marshaller
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
		message = convertToReceivedMessage(marshalledMessage)
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
		Context("normal flow", func() {
			It("should not throw an error", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)

				updateOperationStatusRequest.Status = oc.Status_COMPLETED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
				err := operationContainerHandler(ctx, sampleSettler, message)
				Expect(err).To(BeNil())
			})
			It("should handle oprerationContainer client returning an error", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)

				updateOperationStatusRequest.Status = oc.Status_COMPLETED
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

				returnedErr := errors.New("returned error")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
				err := operationContainerHandler(ctx, sampleSettler, message)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(returnedErr.Error()))
			})
		})
		Context("retries", func() {
			It("should retry 5 times and fail", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)
				updateOperationStatusRequest.Status = oc.Status_IN_PROGRESS

				err := errors.New("Some database error.")
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, err)
				err = operationContainerHandler(ctx, sampleSettler, message)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(buf.String(), "Trying again")).To(Equal(5))
			})
			It("should retry 3 times and succeed", func() {
				operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nil), operationContainerClient, marshaller)
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
				err = operationContainerHandler(ctx, sampleSettler, message)
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
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nonRetryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_CANCELLED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
				})
				It("should handle client error while handling a NonRetryError", func() {
					nonRetryError := &handlerErrors.NonRetryError{
						Message: "NonRetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(nonRetryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("returned error")
					updateOperationStatusRequest.Status = oc.Status_CANCELLED
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(Equal(returnedErr.Error()))
				})
			})
			Context("RetryError", func() {
				It("should handle a RetryError", func() {
					retryError := &handlerErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(retryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, nil)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
				})
				It("should handle client error while handling a RetryError", func() {
					retryError := &handlerErrors.RetryError{
						Message: "RetryError!",
					}
					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(retryError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					returnedErr := errors.New("returned error")
					updateOperationStatusRequest.Status = oc.Status_PENDING
					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, updateOperationStatusRequest).Return(nil, returnedErr)
					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(Equal(returnedErr.Error()))
				})
			})
			Context("default", func() {
				It("should handle a default", func() {
					defaultError := errors.New("default error")

					operationContainerHandler = NewOperationContainerHandler(sampleErrorHandler.SampleErrorHandler(defaultError), operationContainerClient, marshaller)

					operationContainerClient.EXPECT().UpdateOperationStatus(ctx, gomock.Any()).Return(nil, nil)

					err := operationContainerHandler(ctx, sampleSettler, message)
					Expect(err).ToNot(BeNil())
				})
			})
		})
	})
})

func convertToReceivedMessage(msg *azservicebus.Message) *azservicebus.ReceivedMessage {
	var messageID string
	if msg.MessageID != nil {
		messageID = *msg.MessageID
	}

	return &azservicebus.ReceivedMessage{
		ApplicationProperties: msg.ApplicationProperties,
		Body:                  msg.Body,
		ContentType:           msg.ContentType,
		CorrelationID:         msg.CorrelationID,
		MessageID:             messageID,
		PartitionKey:          msg.PartitionKey,
		ReplyTo:               msg.ReplyTo,
		ReplyToSessionID:      msg.ReplyToSessionID,
		ScheduledEnqueueTime:  msg.ScheduledEnqueueTime,
		SessionID:             msg.SessionID,
		Subject:               msg.Subject,
		TimeToLive:            msg.TimeToLive,
		To:                    msg.To,

		// The rest of the fields like LockToken, SequenceNumber, etc., are not present in Message
		// and would need to be mocked or left as zero values if needed.
	}
}
