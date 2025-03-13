package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"log/slog"
	"strings"

	operation "github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LogHandler Suite")
}

var _ = Describe("LogHandler", func() {
	var (
		ctx     context.Context
		buf     bytes.Buffer
		settler shuttle.MessageSettler
		message *azservicebus.ReceivedMessage
		handler shuttle.HandlerFunc
		req     operation.OperationRequest
	)

	BeforeEach(func() {
		buf.Reset()
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
		handler = NewLogHandler(logger, SampleHandler())
	})

	It("should log correctly", func() {
		handler(ctx, settler, message)
		Expect(strings.Count(buf.String(), "LogHandler: ")).To(Equal(2))
	})
	It("should throw an error while unmarshalling", func() {
		invalidMarshalledMessage := &azservicebus.ReceivedMessage{
			Body: []byte(`invalid json`),
		}

		handler(ctx, settler, invalidMarshalledMessage)
		Expect(strings.Count(buf.String(), "LogHandler: ")).To(Equal(3))
		Expect(strings.Count(buf.String(), "Error unmarshalling message")).To(Equal(1))
	})
})

func SampleHandler() shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
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
