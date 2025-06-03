package qos

import (
	"context"
	"log/slog"
	"time"

	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// A QoS handler that is able to log the errors as well.
func NewQosErrorHandler(logger *slog.Logger, errHandler errors.ErrorHandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		if logger == nil {
			logger = ctxlogger.GetLogger(ctx)
		}

		start := time.Now()
		err := errHandler.Handle(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)

		if err != nil {
			logger.With(
				"start_time", start.String(),
				"end_time", t.String(),
				"latency", elapsed.String(),
				"error", err.OriginalError.Error(),
			).Error("QoS: Error occurred in next handler.")
		} else {
			logger.With(
				"start_time", start.String(),
				"end_time", t.String(),
				"latency", elapsed.String(),
			).Info("QoS: Operation processed successfully. No errors returned.")
		}
	}
}
