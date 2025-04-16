package errors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Azure/aks-async/mocks"
	"github.com/Azure/aks-async/runtime/operation"
	sampleHandler "github.com/Azure/aks-async/runtime/testutils/handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ErrorHandler Suite")
}

var _ = Describe("ErrorHandler", func() {
	var (
		ctrl             *gomock.Controller
		ctx              context.Context
		buf              bytes.Buffer
		sampleSettler    shuttle.MessageSettler
		message          *azservicebus.ReceivedMessage
		req              operation.OperationRequest
		mockReceiver     *mocks.MockReceiverInterface
		testErrorMessage error
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		mockReceiver = mocks.NewMockReceiverInterface(ctrl)
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		sampleSettler = &settler.SampleMessageSettler{}
		marshalledOperation, err := json.Marshal(req)
		if err != nil {
			return
		}
		message = &azservicebus.ReceivedMessage{
			Body: marshalledOperation,
		}
	})

	Context("Error handler", func() {
		var (
			handler shuttle.HandlerFunc
		)

		It("should do nothing if no error", func() {
			handler = NewErrorHandler(SampleErrorHandler(nil), mockReceiver, sampleHandler.SampleHandler())
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(0))
		})

		Context("RetryError", func() {
			It("should show RetryError in log", func() {
				testErrorMessage = &RetryError{
					Message: "RetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &RetryError{
					Message: "RetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
			})
		})
		Context("NonRetryError", func() {
			It("should show NonRetryError in log", func() {
				testErrorMessage = &NonRetryError{
					Message: "NonRetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &NonRetryError{
					Message: "NonRetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
			})
		})
		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "Error not recognized")).To(Equal(1))
		})
	})

	Context("Error return handler", func() {
		var (
			errHandler ErrorHandlerFunc
		)

		It("should do nothing if no error", func() {
			errHandler = NewErrorReturnHandler(SampleErrorHandler(nil), mockReceiver, sampleHandler.SampleHandler())
			errHandler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(0))
		})

		Context("RetryError", func() {
			It("should show RetryError in log", func() {
				testErrorMessage = &RetryError{
					Message: "RetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling RetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &RetryError{
					Message: "RetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling RetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
		})

		Context("NonRetryError", func() {
			It("should show NonRetryError in log", func() {
				testErrorMessage = &NonRetryError{
					Message: "NonRetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling NonRetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &NonRetryError{
					Message: "NonRetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling NonRetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
		})

		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, sampleHandler.SampleHandler())
			err := errHandler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "Error not recognized")).To(Equal(1))
			Expect(err).ToNot(BeNil())
		})
	})
})

// Need to re-create this here because importing it from testutils would cause an import cycle error.
func SampleErrorHandler(testErrorMessage error) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		return testErrorMessage
	}
}
