package operation

import (
	"context"
	"time"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"

	"github.com/Azure/aks-async/runtime/entity"
	ec "github.com/Azure/aks-async/runtime/entity_controller"
	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
)

func NewOperationHandler(matcher *matcher.Matcher, hooks []hooks.BaseOperationHooksInterface, entityController ec.EntityController, marshaller shuttle.Marshaller) errors.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *asyncErrors.AsyncError {
		logger := ctxlogger.GetLogger(ctx)

		// 1. Unmarshall the operation
		var body operation.OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			errorMessage := "Error calling unmarshalling message body: " + err.Error()
			logger.Error(errorMessage)
			return &asyncErrors.AsyncError{
				OriginalError: &errors.NonRetryError{Message: "Error unmarshalling message."},
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
			return &asyncErrors.AsyncError{
				OriginalError: &errors.NonRetryError{Message: "Error creating operation instance."},
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

		//TODO(mheberling): Remove this after usage is adopted in Guardrails
		var e entity.Entity
		if entityController != nil {
			e, asyncErr = entityController.GetEntity(ctx, body)
			if asyncErr != nil {
				logger.Error("Something went wrong getting the entity.")
				return asyncErr
			}
		}

		// 4. Guard against concurrency.
		asyncErr = operation.GuardConcurrency(ctx, e)
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
		err = settleMessage(ctx, settler, message, nil)
		if err != nil {
			logger.Error("Settling message: " + err.Error())
			return &asyncErrors.AsyncError{
				OriginalError: err,
				Message:       err.Error(),
				ErrorCode:     500,
			}
		}

		logger.Info("Operation run successfully!")
		return nil
	}
}

func settleMessage(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Settling message.")

	err := settler.CompleteMessage(ctx, message, options)
	if err != nil {
		logger.Error("Unable to settle message.")
		return err
	}

	return nil
}
