package operationsbus

import (
	"context"
	"encoding/json"
	"time"

	sb "github.com/Azure/aks-async/servicebus"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// The processor will be utilized to "process" all the operations by receiving the message, guarding against concurrency, running the operation, and updating the right database status.
func CreateProcessor(
	serviceBusReceiver sb.ReceiverInterface,
	matcher *Matcher,
	operationController OperationController,
	customHandler shuttle.HandlerFunc,
	processorOptions *shuttle.ProcessorOptions,
	hooks []BaseOperationHooksInterface,
) (*shuttle.Processor, error) {

	// Add the operationController hook if the user passed in the operationController
	if operationController != nil {
		operationControllerHook := &OperationControllerHook{
			opController: operationController,
		}
		if hooks == nil {
			hooks = []BaseOperationHooksInterface{
				operationControllerHook,
			}
		} else {
			hooks = append(hooks, operationControllerHook)
		}
	}

	// Define the default handler chain
	defaultHandler := func() shuttle.HandlerFunc {

		// Lock renewal settings
		lockRenewalInterval := 10 * time.Second
		lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

		// Combine handlers into a single default handler
		return shuttle.NewPanicHandler(
			nil,
			shuttle.NewRenewLockHandler(
				lockRenewalOptions,
				NewLogHandler(
					nil,
					NewQosErrorHandler(
						myHandler(matcher, operationController, hooks),
						operationController,
						serviceBusReceiver,
						nil,
					),
				),
			),
		)
	}()

	// Use the default handler if a custom handler is not provided
	if customHandler == nil {
		customHandler = defaultHandler
	}

	if processorOptions == nil {
		processorOptions = &shuttle.ProcessorOptions{
			MaxConcurrency:  1,
			StartMaxAttempt: 5,
		}
	}

	azReceiver, err := serviceBusReceiver.GetAzureReceiver()
	if err != nil {
		return nil, err
	}

	// Create the processor using the (potentially custom) handler
	p := shuttle.NewProcessor(
		azReceiver,
		customHandler,
		processorOptions,
	)

	return p, nil
}

func myHandler(matcher *Matcher, operationController OperationController, hooks []BaseOperationHooksInterface) ErrorHandlerFunc {
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
			return &RetryError{Message: "Error setting operation In Progress"}
		}

		// 4. Get the entity.
		entity, err := operationController.OperationGetEntity(ctx, body)
		if err != nil {
			logger.Error("Entity was not able to be retrieved: " + err.Error())
			return &RetryError{Message: "Error setting operation In Progress"}
		}

		// 5. Guard against concurrency.
		ce := operation.GuardConcurrency(ctx, entity)
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + ce.Err.Error())
			return &RetryError{Message: "Error guarding operation concurrency."}
		}

		// 6. Call run on the operation
		err = operation.Run(ctx)
		if err != nil {
			logger.Error("Something went wrong running the operation: " + err.Error())
			return &RetryError{Message: "Error running operation."}
		}

		// 7. Finish the message
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
