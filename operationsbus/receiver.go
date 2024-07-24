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

func CreateProcessor(serviceBusReceiver sb.ServiceBusReceiver, matcher *Matcher) (*shuttle.Processor, error) {
	//TODO(mheberling): Think if we need to change these time variables.
	lockRenewalInterval := 10 * time.Second
	lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}
	p := shuttle.NewProcessor(serviceBusReceiver.Receiver,
		shuttle.NewPanicHandler(nil,
			shuttle.NewRenewLockHandler(lockRenewalOptions,
				myHandler(matcher))),
		&shuttle.ProcessorOptions{
			MaxConcurrency:  1,
			StartMaxAttempt: 5,
		},
	)

	return p, nil
}

// TODO(mheberling): is there a way to change this so that it doesn't rely only on azure service bus? Maybe try having a message type that has azservicebus.ReceivedMessage insinde and passing that here?
func myHandler(matcher *Matcher) shuttle.HandlerFunc {
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
		operation, err := matcher.CreateInstance(body.OperationName)
		if err != nil {
			logger.Error("Operation type doesn't exist in the matcher: " + err.Error())
			panic(err)
		}

		// 3. Init the operation with the information we have.
		operation.Init(ctx, body)

		// 4. Get the entity.
		entity, err := operation.EntityFetcher(ctx)
		if err != nil {
			logger.Error("Entity was not able to be retrieved: " + err.Error())
			panic(err)
		}

		// 5. Guard against concurrency.
		ce, err := operation.Guardconcurrency(ctx, *entity)
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + err.Error())
			logger.Error("Categorized Error calling GuardConcurrency: " + ce.Error())

			// Retry
			operationRequest := operation.GetOperationRequest(ctx)
			operationRequest.RetryCount++
			retryErr := operation.Retry(ctx, *operationRequest)
			if retryErr != nil {
				logger.Error("Error retrying: " + retryErr.Error())
				panic(retryErr)
			}
			panic(err)
		}

		// 5. Call run on the operation
		operation.Run(ctx)

		// 6. Finish the message
		err = settler.CompleteMessage(ctx, message, nil)
		if err != nil {
			panic(err)
		}
	}
}
