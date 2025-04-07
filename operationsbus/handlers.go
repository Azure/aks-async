package operationsbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	oc "github.com/Azure/OperationContainer/api/v1"
	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
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

func DefaultHandlers(
	serviceBusReceiver sb.ReceiverInterface,
	matcher *Matcher,
	operationContainer oc.OperationContainerClient,
	entityController EntityController,
	logger *slog.Logger,
	hooks []BaseOperationHooksInterface,
) shuttle.HandlerFunc {

	// Lock renewal settings
	lockRenewalInterval := 10 * time.Second
	lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

	var errorHandler ErrorHandlerFunc
	if operationContainer != nil {
		errorHandler = NewOperationContainerHandler(
			NewErrorReturnHandler(
				OperationHandler(matcher, hooks, entityController),
				serviceBusReceiver,
				nil,
			),
			operationContainer,
		)
	} else {
		errorHandler = NewErrorReturnHandler(
			OperationHandler(matcher, hooks, entityController),
			serviceBusReceiver,
			nil,
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
					errorHandler,
				),
			),
		),
	)
}

// A QoS handler that is able to log the errors as well.
func NewQosErrorHandler(errHandler ErrorHandlerFunc) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)
		start := time.Now()
		err := errHandler.Handle(ctx, settler, message)
		t := time.Now()
		elapsed := t.Sub(start)
		logger.Info("QoS: Operation started at: " + start.String() + ". QoS: Operation processed at: " + t.String() + ". QoS: Operation took " + elapsed.String() + " to process.")

		if err != nil {
			logger.Error("QoS: Error ocurred in previousHandler: " + err.Error())
		} else {
			logger.Info("Operation processed successfully. No errors returned.")
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
		logger.Info("QoS: Operation started at: " + start.String() + ". QoS: Operation processed at: " + t.String() + ". QoS: Operation took " + elapsed.String() + " to process.")
	}
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

// Handler for when the user uses the OperationContainer
func NewOperationContainerHandler(errHandler ErrorHandlerFunc, operationContainer oc.OperationContainerClient) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		logger := ctxlogger.GetLogger(ctx)

		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
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
			logger.Error("OperationContainerHandler: Error setting operation in progress: " + err.Error())
			return err
		}

		err = errHandler.Handle(ctx, settler, message)

		if err != nil {
			logger.Info("OperationContainerHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *NonRetryError:
				// Cancel the operation
				logger.Info("OperationContainerHandler: Setting operation as Cancelled.")
				// err = operationContainerOperationCancel(ctx, body.OperationId)
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_CANCELLED,
				}
				_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if err != nil {
					logger.Error("OperationContainerHandler: Something went wrong setting the operation as Cancelled" + err.Error())
					return err
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
					logger.Error("OperationContainerHandler: Something went wrong setting the operation as Pending:" + err.Error())
					return err
				}
			default:
				logger.Info("OperationContainerHandler: Error type not recognized. Operation status not changed.")
			}
		} else {
			logger.Info("Setting Operation as Successful.")
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: body.OperationId,
				Status:      oc.Status_COMPLETED,
			}
			_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			if err != nil {
				logger.Error("OperationContainerHandler: Something went wrong setting the operation as Completed: " + err.Error())
				return err
			}
		}

		return err
	}
}

// NewLogHandler creates a new log handler with the provided logger.
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

func nonRetryOperationError(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Non Retry Operation Error.")

	var body OperationRequest
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

	return nil
}

func OperationHandler(matcher *Matcher, hooks []BaseOperationHooksInterface, entityController EntityController) ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		logger := ctxlogger.GetLogger(ctx)

		// 1. Unmarshall the operation
		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("Error calling unmarshalling message body: " + err.Error())
			return &NonRetryError{Message: "Error unmarshalling message."}
		}

		// 2 Match it with the correct type of operation
		operation, err := matcher.CreateHookedInstace(body.OperationName, hooks)
		if err != nil {
			logger.Error("Operation type doesn't exist in the matcher: " + err.Error())
			return &NonRetryError{Message: "Error creating operation instance."}
		}

		// 3. Init the operation with the information we have.
		_, err = operation.InitOperation(ctx, body)
		if err != nil {
			logger.Error("Something went wrong initializing the operation.")
			return err
		}

		//TODO(mheberling): Remove this after chatting usage is adopted in Guardrails
		var entity Entity
		if entityController != nil {
			entity, err = entityController.GetEntity(ctx, body)
			if err != nil {
				logger.Error("Something went wrong getting the entity.")
				return err
			}
		}

		// 4. Guard against concurrency.
		ce := operation.GuardConcurrency(ctx, entity)
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + ce.Err.Error())
			return err
		}

		// 5. Call run on the operation
		err = operation.Run(ctx)
		if err != nil {
			logger.Error("Something went wrong running the operation: " + err.Error())
			return err
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
