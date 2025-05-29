package operationsbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	oc "github.com/Azure/OperationContainer/api/v1"
	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

type AsyncError struct {
	Message       string
	ErrorCode     int
	RetryAfter    time.Duration
	OriginalError error
}

func (e *AsyncError) Error() string {
	return fmt.Sprintf("AsyncError: Message: %s", e.Message)
}

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
	Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError
}

type ErrorHandlerFunc func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError

func (f ErrorHandlerFunc) Handle(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError {
	return f(ctx, settler, message)
}

func DefaultHandlers(
	serviceBusReceiver sb.ReceiverInterface,
	matcher *Matcher,
	operationContainer oc.OperationContainerClient,
	entityController EntityController,
	logger *slog.Logger,
	hooks []BaseOperationHooksInterface,
	marshaller shuttle.Marshaller,
) shuttle.HandlerFunc {

	// Lock renewal settings
	lockRenewalInterval := 10 * time.Second
	lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

	if marshaller == nil {
		marshaller = &shuttle.DefaultProtoMarshaller{}
	}

	var errorHandler ErrorHandlerFunc
	if operationContainer != nil {
		errorHandler = NewOperationContainerHandler(
			NewErrorReturnHandler(
				OperationHandler(matcher, hooks, entityController, marshaller),
				serviceBusReceiver,
				nil,
				marshaller,
			),
			operationContainer,
			marshaller,
		)
	} else {
		errorHandler = NewErrorReturnHandler(
			OperationHandler(matcher, hooks, entityController, marshaller),
			serviceBusReceiver,
			nil,
			marshaller,
		)
	}

	// Combine handlers into a single default handler
	return shuttle.NewPanicHandler(
		nil,
		shuttle.NewRenewLockHandler(
			lockRenewalOptions,
			NewLogHandler(
				logger,
				NewQosErrorHandler(
					logger,
					errorHandler,
				),
				marshaller,
			),
		),
	)
}

// A QoS handler that is able to log the errors as well.
func NewQosErrorHandler(logger *slog.Logger, errHandler ErrorHandlerFunc) shuttle.HandlerFunc {
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

// NewQoSHandler creates a new QoS handler with the provided logger.
func NewQosHandler(logger *slog.Logger, next shuttle.HandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		if logger == nil {
			logger = ctxlogger.GetLogger(ctx)
		}

		start := time.Now()
		next(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.With(
			"start_time", start.String(),
			"end_time", t.String(),
			"latency", elapsed.String(),
		).Info("QoS: Handler processed.")
	}
}

// An error handler that continues the normal shuttle.HandlerFunc handler chain.
func NewErrorHandler(errHandler ErrorHandlerFunc, receiver sb.ReceiverInterface, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.OriginalError.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(ctx, settler, message, marshaller)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(receiver, ctx, settler, message, marshaller)
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
func NewErrorReturnHandler(errHandler ErrorHandlerFunc, receiver sb.ReceiverInterface, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError {
		err := errHandler.Handle(ctx, settler, message)
		if err != nil {
			logger := ctxlogger.GetLogger(ctx)
			logger.Error("ErrorHandler: Handling error: " + err.Error())
			switch err.OriginalError.(type) {
			case *NonRetryError:
				logger.Info("ErrorHandler: Handling NonRetryError.")
				nonRetryOperationError(ctx, settler, message, marshaller)
			case *RetryError:
				logger.Info("ErrorHandler: Handling RetryError.")
				retryOperationError(receiver, ctx, settler, message, marshaller)
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

// Handler for when the user uses the OperationContainer
func NewOperationContainerHandler(errHandler ErrorHandlerFunc, operationContainer oc.OperationContainerClient, marshaller shuttle.Marshaller) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError {
		logger := ctxlogger.GetLogger(ctx)

		var body OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			logger.Error("OperationContainerHandler: Error unmarshalling message: " + err.Error())
			return nil
		}

		updateOperationStatusRequest := &oc.UpdateOperationStatusRequest{
			OperationId: body.OperationId,
			Status:      oc.Status_IN_PROGRESS,
		}
		_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
		if err != nil {
			errorMessage := "OperationContainerHandler: Error setting operation in progress: " + err.Error()
			logger.Error(errorMessage)
			return &AsyncError{
				OriginalError: err,
				Message:       errorMessage,
				ErrorCode:     500,
			}
		}

		err = errHandler.Handle(ctx, settler, message)

		if err != nil {
			logger.Info("OperationContainerHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				// Fail the operation
				logger.Info("OperationContainerHandler: Setting operation as Failed.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_FAILED,
				}
				_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if err != nil {
					errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Failed" + err.Error()
					logger.Error(errorMessage)
					return &AsyncError{
						OriginalError: err,
						Message:       errorMessage,
						ErrorCode:     500,
					}
				}
			case *RetryError:
				// Set the operation as Pending
				logger.Info("OperationContainerHandler: Setting operation as Pending.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_PENDING,
				}
				_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if err != nil {
					errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Pending:" + err.Error()
					logger.Error(errorMessage)
					return &AsyncError{
						OriginalError: err,
						Message:       errorMessage,
						ErrorCode:     500,
					}
				}
			default:
				errorMessage := "OperationContainerHandler: Error type not recognized. Operation status not changed."
				logger.Info(errorMessage)
				return &AsyncError{
					OriginalError: err,
					Message:       errorMessage,
					ErrorCode:     500,
				}
			}
		} else {
			logger.Info("Setting Operation as Successful.")
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: body.OperationId,
				Status:      oc.Status_COMPLETED,
			}
			_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			if err != nil {
				errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Completed:" + err.Error()
				logger.Error(errorMessage)
				return &AsyncError{
					OriginalError: err,
					Message:       errorMessage,
					ErrorCode:     500,
				}
			}
		}

		return nil
	}
}

// NewLogHandler creates a new log handler with the provided logger.
func NewLogHandler(logger *slog.Logger, next shuttle.HandlerFunc, marshaller shuttle.Marshaller) shuttle.HandlerFunc {
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
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			logger.Error("LogHandler: Error unmarshalling message:" + err.Error())
		}

		logger.Info("LogHandler: OperationId: " + body.OperationId)

		next(newCtx, settler, message)
	}
}

func nonRetryOperationError(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, marshaller shuttle.Marshaller) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Non Retry Operation Error.")

	var body OperationRequest
	err := marshaller.Unmarshal(message.Message(), &body)
	if err != nil {
		logger.Error("Error calling ReceiveOperation: " + err.Error())
		return err
	}

	// Settle message
	deadLetterMessage(ctx, settler, message, nil)

	return nil
}

func retryOperationError(receiver sb.ReceiverInterface, ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, marshaller shuttle.Marshaller) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Abandoning message for retry.")

	azReceiver, err := receiver.GetAzureReceiver()
	if err != nil {
		return err
	}

	var body OperationRequest
	err = marshaller.Unmarshal(message.Message(), &body)
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

func OperationHandler(matcher *Matcher, hooks []BaseOperationHooksInterface, entityController EntityController, marshaller shuttle.Marshaller) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *AsyncError {
		logger := ctxlogger.GetLogger(ctx)

		// 1. Unmarshall the operation
		var body OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			errorMessage := "Error calling unmarshalling message body: " + err.Error()
			logger.Error(errorMessage)
			return &AsyncError{
				OriginalError: &NonRetryError{Message: "Error unmarshalling message."},
				Message:       errorMessage,
				ErrorCode:     500,
				RetryAfter:    0 * time.Second,
			}
		}

		// 2 Match it with the correct type of operation
		operation, err := matcher.CreateHookedInstace(body.OperationName, hooks)
		if err != nil {
			errorMessage := "Operation type doesn't exist in the matcher: " + err.Error()
			logger.Error(errorMessage)
			return &AsyncError{
				OriginalError: &NonRetryError{Message: "Error creating operation instance."},
				Message:       errorMessage,
				ErrorCode:     500,
				RetryAfter:    0 * time.Second,
			}
		}

		// 3. Init the operation with the information we have.
		_, asyncErr := operation.InitOperation(ctx, body)
		if asyncErr != nil {
			logger.Error("Something went wrong initializing the operation.")
			return asyncErr
		}

		//TODO(mheberling): Remove this after chatting usage is adopted in Guardrails
		//TODO(mheberling): Look at using pointers here instead of value since we're using proto.
		var entity Entity
		if entityController != nil {
			entity, asyncErr = entityController.GetEntity(ctx, body)
			if asyncErr != nil {
				logger.Error("Something went wrong getting the entity.")
				return asyncErr
			}
		}

		// 4. Guard against concurrency.
		asyncErr = operation.GuardConcurrency(ctx, entity)
		if asyncErr != nil {
			logger.Error("Error calling GuardConcurrency: " + asyncErr.Error())
			return asyncErr
		}

		// 5. Call run on the operation
		asyncErr = operation.Run(ctx)
		if asyncErr != nil {
			logger.Error("Something went wrong running the operation: " + asyncErr.Error())
			return asyncErr
		}

		// 6. Finish the message
		settleMessage(ctx, settler, message, nil)

		logger.Info("Operation run successfully!")
		return nil
	}
}

func settleMessage(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Settling message.")

	err := settler.CompleteMessage(ctx, message, options)
	if err != nil {
		logger.Error("Unable to settle message.")
	}
}

func deadLetterMessage(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("DeadLettering message.")

	err := settler.DeadLetterMessage(ctx, message, options)
	if err != nil {
		logger.Error("Unable to deadletter message.")
	}
}
