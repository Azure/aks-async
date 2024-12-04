package operationsbus

import (
	"context"
	"encoding/json"
	"log/slog"
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
	logger *slog.Logger,
	customHandler shuttle.HandlerFunc,
	processorOptions *shuttle.ProcessorOptions,
	hooks []BaseOperationHooksInterface,
) (*shuttle.Processor, error) {

	// Define the default handler chain
	defaultHandler := func() shuttle.HandlerFunc {

		// Lock renewal settings
		lockRenewalInterval := 10 * time.Second
		lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

		var errorHandler ErrorHandlerFunc
		if operationController != nil {
			errorHandler = NewOperationControllerHandler(
				NewErrorReturnHandler(
					myHandler(matcher, hooks),
					serviceBusReceiver,
					nil,
				),
				operationController,
			)
		} else {
			errorHandler = NewErrorReturnHandler(
				myHandler(matcher, hooks),
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

func myHandler(matcher *Matcher, hooks []BaseOperationHooksInterface) ErrorHandlerFunc {
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

		// 4. Guard against concurrency.
		ce := operation.GuardConcurrency(ctx)
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + ce.Err.Error())
			return &RetryError{Message: "Error guarding operation concurrency."}
		}

		// 5. Call run on the operation
		err = operation.Run(ctx)
		if err != nil {
			logger.Error("Something went wrong running the operation: " + err.Error())
			return &RetryError{Message: "Error running operation."}
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
