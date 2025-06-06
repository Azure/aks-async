package log

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// Creates a new log handler with the provided logger and provides message logging.
func NewLogHandler(logger *slog.Logger, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		if logger == nil {
			logger = ctxlogger.GetLogger(ctx)
			ctx = ctxlogger.WithLogger(ctx, logger)
		}
		if marshaller == nil {
			marshaller = &shuttle.DefaultProtoMarshaller{}
		}

		//TODO(mheberling): Set these as attributes, not individual strings.
		logger.Info("LogHandler: Delivery count: " + fmt.Sprint(message.DeliveryCount))
		if message.CorrelationID != nil {
			logger.Info("LogHandler: Corrolation Id: " + *message.CorrelationID)
		}

		var body operation.OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			logger.Error("LogHandler: Error unmarshalling message:" + err.Error())
		}

		logger.Info("LogHandler: OperationId: " + body.OperationId)

		next(ctx, settler, message)
	}
}
