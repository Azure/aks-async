package operationsbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// Default errors for the error handler.
type RetryError struct {
	Message string
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("RetryError: %s", e.Message)
}

type NonRetryError struct {
	Message string
}

func (e *NonRetryError) Error() string {
	return fmt.Sprintf("NonRetryError: %s", e.Message)
}

// ErrorHandler interface that returns an error. Required for any error handling and not depending on panics.
type ErrorHandler interface {
	Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error
}

type ErrorHandlerFunc func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error

func (f ErrorHandlerFunc) Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	return f(ctx, settler, message)
}

func NewErrorHandler(errHandler ErrorHandlerFunc, operationController OperationController, receiver sb.ReceiverInterface, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(operationController, ctx, settler, message)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(operationController, receiver, ctx, settler, message)
			default:
				logger.Info("Error handled: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}
	}
}

func NewQosErrorHandler(errHandler ErrorHandlerFunc, operationController OperationController, receiver sb.ReceiverInterface, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)
		start := time.Now()
		err := errHandler.Handle(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.Info("QoS: Operation started at: " + start.String())
		logger.Info("QoS: Operation processed at: " + t.String())
		logger.Info("QoS: Operation took " + elapsed.String() + " to process.")
		if err != nil {
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(operationController, ctx, settler, message)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(operationController, receiver, ctx, settler, message)
			default:
				logger.Info("Error handled: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}
	}
}

// NewLogHandler creates a new log handler with the provided logger
func NewLogHandler(logger *slog.Logger, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		if logger == nil {
			logger = ctxlogger.GetLogger(ctx)
		}
		newCtx := ctxlogger.WithLogger(ctx, logger)

		logger.Info("LogHandler: Delivery count: " + fmt.Sprint(message.DeliveryCount))
		if message.CorrelationID != nil {
			logger.Info("LogHandler: Corrolation Id: " + *message.CorrelationID)
		}

		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("LogHandler: Error unmarshalling message:" + err.Error())
		}

		logger.Info("LogHandler: OperationId: " + body.OperationId)

		next(newCtx, settler, message)
	}
}

// NewQoSHandler creates a new QoS handler with the provided logger
func NewQoSHandler(logger *slog.Logger, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		if logger == nil {
			logger = ctxlogger.GetLogger(ctx)
		}

		start := time.Now()
		next(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.Info("QoSHandler: Operation started at: " + start.String())
		logger.Info("QoSHandler: Operation processed at: " + t.String())
		logger.Info("QoSHandler: Operation took " + elapsed.String() + " to process.")
	}
}

func nonRetryOperationError(operationController OperationController, ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Non Retry Operation Error.")

	var body OperationRequest
	err := json.Unmarshal(message.Body, &body)
	if err != nil {
		logger.Error("Error calling ReceiveOperation: " + err.Error())
		return err
	}

	// Settle message
	settleMessage(ctx, settler, message, nil)

	// Cancel the operation
	err = operationController.OperationCancel(ctx, body.OperationId)
	if err != nil {
		logger.Error("Something went wrong setting the operation as Cancelled.")
		return err
	}
	return nil
}

func retryOperationError(operationController OperationController, receiver sb.ReceiverInterface, ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Abandoning message for retry.")

	azReceiver, err := receiver.GetAzureReceiver()
	if err != nil {
		return err
	}

	var body OperationRequest
	err = json.Unmarshal(message.Body, &body)
	if err != nil {
		logger.Error("Error calling ReceiveOperation: " + err.Error())
		return err
	}

	// Retry the message
	err = azReceiver.AbandonMessage(ctx, message, nil)
	if err != nil {
		logger.Error("Error abandoning message: " + err.Error())
		return err
	}

	// Set the operation as Pending
	err = operationController.OperationPending(ctx, body.OperationId)
	if err != nil {
		logger.Error("Something went wrong setting the operation as Pending.")
		return err
	}

	return nil
}
