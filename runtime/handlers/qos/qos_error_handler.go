package qos

import (
	"context"
	"time"

	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// A QoS handler that is able to log the errors as well.
func NewQosErrorHandler(errHandler errors.ErrorHandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)

		start := time.Now()
		err := errHandler.Handle(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.Info("QoSErrorHandler: Operation started at: " + start.String() + ", processed at: " + t.String() + ", and processed in: " + elapsed.String())

		if err != nil {
			logger.Error("QoSErrorHandler: Error ocurred in previousHandler: " + err.Error())
		}
	}
}
