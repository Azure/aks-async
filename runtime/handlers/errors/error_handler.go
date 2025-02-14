package errors

import (
	"context"
	"encoding/json"

	"github.com/Azure/aks-async/runtime/operation"
	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// ErrorHandler interface that returns an error. Required for any error handling and not depending on panics.
type ErrorHandler interface {
	Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error
}

type ErrorHandlerFunc func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error

func (f ErrorHandlerFunc) Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	return f(ctx, settler, message)
}

// An error handler that continues the normal shuttle.HandlerFunc handler chain.
func NewErrorHandler(errHandler ErrorHandlerFunc, receiver sb.ReceiverInterface, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(ctx, settler, message)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(receiver, ctx, settler, message)
			default:
				logger.Info("Error handled: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}
	}
}

// An error handler that provides the error to the parent handler for logging.
func NewErrorReturnHandler(errHandler ErrorHandlerFunc, receiver sb.ReceiverInterface, next shuttle.HandlerFunc) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(ctx, settler, message)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(receiver, ctx, settler, message)
			default:
				logger.Info("Error handled: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}

		return err
	}
}

func nonRetryOperationError(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Non Retry Operation Error.")

	var body operation.OperationRequest
	err := json.Unmarshal(message.Body, &body)
	if err != nil {
		logger.Error("Error calling ReceiveOperation: " + err.Error())
		return err
	}

	// Settle message
	deadLetterMessage(ctx, settler, message, nil)

	return nil
}

func retryOperationError(receiver sb.ReceiverInterface, ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Abandoning message for retry.")

	azReceiver, err := receiver.GetAzureReceiver()
	if err != nil {
		return err
	}

	var body operation.OperationRequest
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

	return nil
}

func deadLetterMessage(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("DeadLettering message.")

	err := settler.DeadLetterMessage(ctx, message, options)
	if err != nil {
		logger.Error("Unable to deadletter message.")
	}
}
