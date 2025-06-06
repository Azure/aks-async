package errors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/operation"
	sampleHandler "github.com/Azure/aks-async/runtime/testutils/handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ErrorHandler Suite")
}

var _ = Describe("ErrorHandler", func() {
	var (
		ctx              context.Context
		buf              bytes.Buffer
		sampleSettler    shuttle.MessageSettler
		message          *azservicebus.ReceivedMessage
		req              *operation.OperationRequest
		testErrorMessage error
		marshaller       shuttle.Marshaller
	)

	BeforeEach(func() {
		buf.Reset()
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

		sampleSettler = &settler.SampleMessageSettler{}

		marshaller = &shuttle.DefaultProtoMarshaller{}
		req = &operation.OperationRequest{
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
		marshalledMessage, err := marshaller.Marshal(req)
		if err != nil {
			return
		}
		message = convert.ConvertToReceivedMessage(marshalledMessage)
	})

	Context("Error handler", func() {
		var (
			handler shuttle.HandlerFunc
		)

		It("should do nothing if no error", func() {
			handler = NewErrorHandler(SampleErrorHandler(nil), sampleHandler.SampleHandler(), marshaller)
			handler(ctx, sampleSettler, message)
			fmt.Println(buf.String())
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(0))
		})

		Context("RetryError", func() {
			It("should show RetryError in log", func() {
				testErrorMessage = &asyncErrors.RetryError{
					Message: "RetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &asyncErrors.RetryError{
					Message: "RetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
			})
		})
		Context("NonRetryError", func() {
			It("should show NonRetryError in log", func() {
				testErrorMessage = &asyncErrors.NonRetryError{
					Message: "NonRetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &asyncErrors.NonRetryError{
					Message: "NonRetryError",
				}
				handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				handler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
			})
		})
		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
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
			errHandler = NewErrorReturnHandler(SampleErrorHandler(nil), sampleHandler.SampleHandler(), marshaller)
			errHandler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(0))
		})

		Context("RetryError", func() {
			It("should show RetryError in log", func() {
				testErrorMessage = &asyncErrors.RetryError{
					Message: "RetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling RetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &asyncErrors.RetryError{
					Message: "RetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling RetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
		})

		Context("NonRetryError", func() {
			It("should show NonRetryError in log", func() {
				testErrorMessage = &asyncErrors.NonRetryError{
					Message: "NonRetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling NonRetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
			It("should handle settler error", func() {
				failureContentType := "failure_test"
				message.ContentType = &failureContentType
				testErrorMessage = &asyncErrors.NonRetryError{
					Message: "NonRetryError",
				}
				errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
				err := errHandler(ctx, sampleSettler, message)
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(3))
				Expect(strings.Count(buf.String(), "ErrorReturnHandler: Handling NonRetryError")).To(Equal(1))
				Expect(err).ToNot(BeNil())
			})
		})

		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), sampleHandler.SampleHandler(), marshaller)
			err := errHandler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "ErrorReturnHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "Error not recognized")).To(Equal(1))
			Expect(err).ToNot(BeNil())
		})
	})
})

// Need to re-create this here because importing it from testutils would cause an import cycle error.
func SampleErrorHandler(testErrorMessage error) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *asyncErrors.AsyncError {
		if testErrorMessage != nil {
			return &asyncErrors.AsyncError{OriginalError: testErrorMessage}
		}
		return nil
	}
}
