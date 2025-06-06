package qos

import (
	"bytes"
	"context"

	"log/slog"
	"strings"

	operation "github.com/Azure/aks-async/runtime/operation"
	sampleHandler "github.com/Azure/aks-async/runtime/testutils/handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
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
		req           *operation.OperationRequest
		marshaller    shuttle.Marshaller
	)

	BeforeEach(func() {
		buf.Reset()
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)

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
		sampleSettler = &settler.SampleMessageSettler{}
		marshaller = &shuttle.DefaultProtoMarshaller{}
		marshalledMessage, err := marshaller.Marshal(req)
		if err != nil {
			return
		}
		message = convert.ConvertToReceivedMessage(marshalledMessage)
		handler = NewQosHandler(nil, sampleHandler.SampleHandler())
	})

	It("should have right number of logs", func() {
		handler(ctx, sampleSettler, message)
		Expect(strings.Count(buf.String(), "QoS: ")).To(Equal(1))
	})
})
