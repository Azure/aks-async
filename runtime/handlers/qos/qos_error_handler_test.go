package qos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"log/slog"
	"strings"

	operation "github.com/Azure/aks-async/runtime/operation"
	sampleErrorHandler "github.com/Azure/aks-async/runtime/testutils/error_handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QoSErrorHandler", func() {
	var (
		ctx           context.Context
		buf           bytes.Buffer
		sampleSettler shuttle.MessageSettler
		message       *azservicebus.ReceivedMessage
		handler       shuttle.HandlerFunc
		req           operation.OperationRequest
	)

	BeforeEach(func() {
		buf.Reset()
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

	It("should have right count of logs", func() {
		handler = NewQosErrorHandler(sampleErrorHandler.SampleErrorHandler(nil))
		handler(ctx, sampleSettler, message)
		Expect(strings.Count(buf.String(), "QoSErrorHandler: ")).To(Equal(1))
	})

	It("should log error in next handler", func() {
		err := errors.New("Random error")
		handler = NewQosErrorHandler(sampleErrorHandler.SampleErrorHandler(err))
		handler(ctx, sampleSettler, message)
		Expect(strings.Count(buf.String(), "QoSErrorHandler: ")).To(Equal(2))
	})
})
