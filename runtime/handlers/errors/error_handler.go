package errors

import (
	"context"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// ErrorHandler interface that returns an error. Required for any error handling accross handlers.
type ErrorHandler interface {
	Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error
}

type ErrorHandlerFunc func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error

func (f ErrorHandlerFunc) Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	return f(ctx, settler, message)
}

// An error handler that continues the normal shuttle.HandlerFunc handler chain.
func NewErrorHandler(errHandler ErrorHandlerFunc, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())

			var actionErr error
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				actionErr = nonRetryOperationError(ctx, settler, message, marshaller)
				if actionErr != nil {
					logger.Error("ErrorHandler: " + actionErr.Error())
				}
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				actionErr = retryOperationError(ctx, settler, message, marshaller)
				if actionErr != nil {
					logger.Error("ErrorHandler: " + actionErr.Error())
				}
			default:
				logger.Info("ErrorHandler: Error not recognized: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}
	}
}

// An error handler that provides the error to the parent handler for logging.
func NewErrorReturnHandler(errHandler ErrorHandlerFunc, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorReturnHandler: Handling error: " + err.Error())

			var actionErr error
			switch err.(type) {
			case *NonRetryError:
				logger.Info("ErrorReturnHandler: Handling NonRetryError.")
				actionErr = nonRetryOperationError(ctx, settler, message, marshaller)
				if actionErr != nil {
					logger.Error("ErrorReturnHandler: " + actionErr.Error())
					return actionErr
				}
			case *RetryError:
				logger.Info("ErrorReturnHandler: Handling RetryError.")
				actionErr = retryOperationError(ctx, settler, message, marshaller)
				if actionErr != nil {
					logger.Error("ErrorReturnHandler: " + actionErr.Error())
					return actionErr
				}
			default:
				logger.Info("ErrorReturnHandler: Error not recognized: " + err.Error())
			}
		}

		if next != nil {
			next(ctx, settler, message)
		}

		return err
	}
}

func nonRetryOperationError(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, marshaller shuttle.Marshaller) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Non Retry Operation Error.")

	err := settler.DeadLetterMessage(ctx, message, nil)
	if err != nil {
		logger.Error("Unable to deadletter message: " + err.Error())
		return err
	}

	return nil
}

func retryOperationError(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, marshaller shuttle.Marshaller) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Abandoning message for retry.")

	err := settler.AbandonMessage(ctx, message, nil)
	if err != nil {
		logger.Error("Error abandoning message: " + err.Error())
		return err
	}

	return nil
}
