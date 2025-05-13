package qos

import (
	"context"
	"time"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// NewQoSHandler creates a new QoS handler with the provided logger.
func NewQoSHandler(next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)

		start := time.Now()
		next(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.Info("QoSHandler: Operation started at: " + start.String() + ", processed at: " + t.String() + ", and processed in: " + elapsed.String())
	}
}
