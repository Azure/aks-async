package qos

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"log/slog"
	"strings"

	operation "github.com/Azure/aks-async/runtime/operation"
	sampleErrorHandler "github.com/Azure/aks-async/runtime/testutils/error_handler"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-async/runtime/testutils/toolkit/convert"
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
	})

	It("should have right count of logs with no error", func() {
		handler = NewQosErrorHandler(nil, sampleErrorHandler.SampleErrorHandler(nil))
		handler(ctx, sampleSettler, message)
		fmt.Println(buf.String())
		Expect(strings.Count(buf.String(), "QoS: ")).To(Equal(1))
		Expect(strings.Count(buf.String(), "error")).To(Equal(0))
	})

	It("should log error in next handler", func() {
		err := errors.New("Random error")
		handler = NewQosErrorHandler(nil, sampleErrorHandler.SampleErrorHandler(err))
		handler(ctx, sampleSettler, message)
		fmt.Println(buf.String())
		Expect(strings.Count(buf.String(), "QoS: ")).To(Equal(1))
		Expect(strings.Count(buf.String(), "error=")).To(Equal(1))
	})
})
