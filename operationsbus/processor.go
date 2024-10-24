package operationsbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sb "github.com/Azure/aks-async/servicebus"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// The processor will be utilized to "process" all the operations by receiving the message, guarding against concurrency, running the operation, and updating the right database status.
func CreateProcessor(
	sender sb.SenderInterface,
	serviceBusReceiver sb.ReceiverInterface,
	matcher *Matcher,
	operationController OperationController,
	customHandler shuttle.HandlerFunc,
	processorOptions *shuttle.ProcessorOptions,
	hooks []BaseOperationHooksInterface,
) (*shuttle.Processor, error) {

	// Define the default handler chain
	defaultHandler := func() shuttle.HandlerFunc {
		// Default panic handler
		panicOptions := &shuttle.PanicHandlerOptions{
			OnPanicRecovered: basicPanicRecovery(operationController),
		}

		// Lock renewal settings
		lockRenewalInterval := 10 * time.Second
		lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

		// Combine handlers into a single default handler
		return shuttle.NewPanicHandler(
			panicOptions,
			shuttle.NewRenewLockHandler(
				lockRenewalOptions,
				myHandler(matcher, operationController, sender, hooks),
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
		&shuttle.ProcessorOptions{
			MaxConcurrency:  1,
			StartMaxAttempt: 5,
		},
	)

	return p, nil
}

// TODO(mheberling): is there a way to change this so that it doesn't rely only on azure service bus? Maybe try having a message type that has azservicebus.ReceivedMessage inside and passing that here?
func myHandler(matcher *Matcher, operationController OperationController, sender sb.SenderInterface, hooks []BaseOperationHooksInterface) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)

		// 1. Unmarshall the operatoin
		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("Error calling ReceiveOperation: " + err.Error())
			panic(err)
		}

		if body.RetryCount >= 10 {
			logger.Error("Operation has passed the retry limit.")
			panic(errors.New(fmt.Sprintf("Operation has retried %d times. The limit is 10.", body.RetryCount)))
		}

		// 2 Match it with the correct type of operation
		operation, err := matcher.CreateHookedInstace(body.OperationName, hooks)
		if err != nil {
			logger.Error("Operation type doesn't exist in the matcher: " + err.Error())
			panic(err)
		}

		panicOptions := &shuttle.PanicHandlerOptions{
			OnPanicRecovered: operationPanicRecovery(operationController, sender),
		}

		// We add a different panic handler here because we only want to retry the operation if: we are able to unmarshall the message,
		// the retry limit has not been exceeded, and wee were able to instantiate the operation. These 3 things will not change by
		// returning the message to the queue so we don't want to retry.
		handler := shuttle.NewPanicHandler(panicOptions, shuttle.HandlerFunc(func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {

			// Set the operation as in progress.
			err = operationController.OperationInProgress(ctx, body.OperationId)
			if err != nil {
				logger.Error("Something went wrong setting operation In Progress.")
				panic(err)
			}

			// 3. Init the operation with the information we have.
			_, err = operation.InitOperation(ctx, body)
			if err != nil {
				logger.Error("Something went wrong initializing the operation.")
				panic(err)
			}

			// 4. Get the entity.
			entity, err := operationController.OperationGetEntity(ctx, body)
			if err != nil {
				logger.Error("Entity was not able to be retrieved: " + err.Error())
				panic(err)
			}

			// 5. Guard against concurrency.
			ce := operation.GuardConcurrency(ctx, entity)
			if err != nil {
				logger.Error("Error calling GuardConcurrency: " + ce.Err.Error())
				panic(ce.Err)
			}

			// 6. Call run on the operation
			err = operation.Run(ctx)
			if err != nil {
				logger.Error("Something went wrong running the operation: " + err.Error())
				panic(err)
			}

			// Set operation as FINISHED
			err = operationController.OperationCompleted(ctx, body.OperationId)
			if err != nil {
				logger.Error("Something went wrong setting the operation as Completed.")
				panic(err)
			}

			// 7. Finish the message
			settleMessage(ctx, settler, message, nil)

			logger.Info("Operation run successfully!")
		}))

		handler.Handle(ctx, settler, message)
	}
}

func basicPanicRecovery(operationController OperationController) func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, recovered any) {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, recovered any) {
		logger := ctxlogger.GetLogger(ctx)
		logger.Info("Recovering from panic before getting operation.")

		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("Error calling ReceiveOperation: " + err.Error())
			return
		}

		// Settle message
		settleMessage(ctx, settler, message, nil)

		// Cancel the operation
		err = operationController.OperationCancel(ctx, body.OperationId)
		if err != nil {
			logger.Error("Something went wrong setting the operation as Cancelled.")
		}
	}
}

func operationPanicRecovery(operationController OperationController, sender sb.SenderInterface) func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, recovered any) {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage, recovered any) {
		logger := ctxlogger.GetLogger(ctx)
		logger.Info("Recovering from panic after getting operation.")

		var body OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("Error calling ReceiveOperation: " + err.Error())
			return
		}
		// Retry the message
		//TODO(mheberling): Remove retry logic in favor of servicebus automatic retry.
		err = body.Retry(ctx, sender)
		if err != nil {
			logger.Error("Error retrying: " + err.Error())
		}

		// Settle message
		settleMessage(ctx, settler, message, nil)

		// Set the operation as Pending
		err = operationController.OperationPending(ctx, body.OperationId)
		if err != nil {
			logger.Error("Something went wrong setting the operation as Pending.")
		}
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
