package qos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"log/slog"
	"strings"

	handlerError "github.com/Azure/aks-async/runtime/handlers/errors"
	operation "github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QoSErrorHandler", func() {
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
		// ctrl = gomock.NewController(GinkgoT())
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

	It("should have right count of logs", func() {
		handler = NewQosErrorHandler(SampleErrorHandler(nil))
		handler(ctx, settler, message)
		Expect(strings.Count(buf.String(), "QoSErrorHandler: ")).To(Equal(1))
	})

	It("should log error in next handler", func() {
		err := errors.New("Random error")
		handler = NewQosErrorHandler(SampleErrorHandler(err))
		handler(ctx, settler, message)
		Expect(strings.Count(buf.String(), "QoSErrorHandler: ")).To(Equal(2))
	})
})

func SampleErrorHandler(testError error) handlerError.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		return testError
	}
}
