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

var _ = Describe("DeadLetterQueueHandler", func() {
	var (
		ctrl             *gomock.Controller
		ctx              context.Context
		buf              bytes.Buffer
		settler          shuttle.MessageSettler
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

		settler = &fakeMessageSettler{}
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
		It("should show RetryError in log", func() {
			testErrorMessage = &RetryError{
				Message: "RetryError",
			}
			handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
		})
		It("should show NonRetryError in log", func() {
			testErrorMessage = &NonRetryError{
				Message: "NonRetryError",
			}
			handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
		})
		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			handler = NewErrorHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			handler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(1))
			Expect(strings.Count(buf.String(), "Error handled: ")).To(Equal(1))
		})
	})

	Context("Error return handler", func() {
		var (
			errHandler ErrorHandlerFunc
		)
		It("should show RetryError in log", func() {
			testErrorMessage = &RetryError{
				Message: "RetryError",
			}
			errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			err := errHandler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "ErrorHandler: Handling RetryError")).To(Equal(1))
			Expect(err).ToNot(BeNil())
		})
		It("should show NonRetryError in log", func() {
			testErrorMessage = &NonRetryError{
				Message: "NonRetryError",
			}
			errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			err := errHandler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(2))
			Expect(strings.Count(buf.String(), "ErrorHandler: Handling NonRetryError")).To(Equal(1))
			Expect(err).ToNot(BeNil())
		})
		It("should show different error in log", func() {
			testErrorMessage = errors.New("Random error")
			errHandler = NewErrorReturnHandler(SampleErrorHandler(testErrorMessage), mockReceiver, SampleHandler())
			err := errHandler(ctx, settler, message)
			Expect(strings.Count(buf.String(), "ErrorHandler: ")).To(Equal(1))
			Expect(strings.Count(buf.String(), "Error handled: ")).To(Equal(1))
			Expect(err).ToNot(BeNil())
		})
	})
})

func SampleHandler() shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
	}
}

func SampleErrorHandler(testErrorMessage error) ErrorHandlerFunc {
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
