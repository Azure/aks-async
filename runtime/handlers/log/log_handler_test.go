package log

import (
	"bytes"
	"context"
	"testing"

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

func TestLogHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LogHandler Suite")
}

var _ = Describe("LogHandler", func() {
	var (
		ctx             context.Context
		buf             bytes.Buffer
		sampleSettler   shuttle.MessageSettler
		receivedMessage *azservicebus.ReceivedMessage
		handler         shuttle.HandlerFunc
		req             *operation.OperationRequest
		marshaller      shuttle.Marshaller
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
		message, err := marshaller.Marshal(req)
		if err != nil {
			return
		}
		receivedMessage = convertToReceivedMessage(message)
		handler = NewLogHandler(logger, sampleHandler.SampleHandler(), marshaller)
	})

	It("should log correctly", func() {
		handler(ctx, sampleSettler, receivedMessage)
		Expect(strings.Count(buf.String(), "LogHandler: ")).To(Equal(2))
	})
	It("should throw an error while unmarshalling", func() {
		invalidMarshalledMessage := &azservicebus.ReceivedMessage{
			Body: []byte(`invalid json`),
		}

		handler(ctx, sampleSettler, invalidMarshalledMessage)
		Expect(strings.Count(buf.String(), "LogHandler: ")).To(Equal(3))
		Expect(strings.Count(buf.String(), "Error unmarshalling message")).To(Equal(1))
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
