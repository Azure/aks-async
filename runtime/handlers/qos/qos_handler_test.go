package qos

import (
	"bytes"
	"context"
	"encoding/json"

	"log/slog"
	"strings"

	operation "github.com/Azure/aks-async/runtime/operation"
	sampleHandler "github.com/Azure/aks-async/runtime/testutils/handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QoSHandler", func() {
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
		// ctrl = gomock.NewController(GinkgoT())
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
		handler = NewQoSHandler(sampleHandler.SampleHandler())
	})

	It("should have right number of logs", func() {
		handler(ctx, sampleSettler, message)
		Expect(strings.Count(buf.String(), "QoSHandler: ")).To(Equal(1))
	})
})
