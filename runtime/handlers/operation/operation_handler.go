package operation

import (
	"context"
	"encoding/json"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"

	"github.com/Azure/aks-async/runtime/entity"
	ec "github.com/Azure/aks-async/runtime/entity_controller"
	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
)

func NewOperationHandler(matcher *matcher.Matcher, hooks []hooks.BaseOperationHooksInterface, entityController ec.EntityController) errors.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		logger := ctxlogger.GetLogger(ctx)

		// 1. Unmarshall the operation
		var body operation.OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("Error calling unmarshalling message body: " + err.Error())
			return &errors.NonRetryError{Message: "Error unmarshalling message."}
		}

		// 2 Match it with the correct type of operation
		operation, err := matcher.CreateHookedInstace(body.OperationName, hooks)
		if err != nil {
			logger.Error("Operation type doesn't exist in the matcher: " + err.Error())
			return &errors.NonRetryError{Message: "Error creating operation instance."}
		}

		// 3. Init the operation with the information we have.
		_, err = operation.InitOperation(ctx, body)
		if err != nil {
			logger.Error("Something went wrong initializing the operation.")
			return &errors.RetryError{Message: "Error setting operation In Progress"}
		}

		//TODO(mheberling): Remove this after usage is adopted in Guardrails
		var operationEntity entity.Entity
		if entityController != nil {
			operationEntity, err = entityController.GetEntity(ctx, body)
			if err != nil {
				logger.Error("Something went wrong getting the entity.")
				return &errors.RetryError{Message: "Error getting operationEntity"}
			}
		}

		// 4. Guard against concurrency.
		ce := operation.GuardConcurrency(ctx, operationEntity)
		if ce != nil {
			logger.Error("Error calling GuardConcurrency: " + ce.Err.Error())
			return &errors.RetryError{Message: "Error guarding operation concurrency."}
		}

		// 5. Call run on the operation
		err = operation.Run(ctx)
		if err != nil {
			logger.Error("Something went wrong running the operation: " + err.Error())
			return &errors.RetryError{Message: "Error running operation."}
		}

		// 6. Finish the message
		err = settleMessage(ctx, settler, message, nil)
		if err != nil {
			logger.Error("Settling message: " + err.Error())
			return err
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
