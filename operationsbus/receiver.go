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

// The receiver interface is what the user who is implementing their own async component (what actually receives and runs the operations) will need to implement, to do with their own definitions.
// type ReceiverInterface interface {
// 	ReceiveOperation(context.Context) (APIOperation, error)
// }
//
// type Receiver struct {
// 	//TODO(mheberling): Change this to have a long lived connection to the receiver and sender as well.
// 	sbusClient *azservicebus.Client //TODO(mheberling): change to an interface so that they can change to use any service bus client not just azure.
// 	ReceiverInterface
// }

// func NewReceiver(sbusClient *azservicebus.Client) *Receiver {
// 	receiver := &Receiver{
// 		sbusClient: sbusClient,
// 	}
// 	return receiver
// }

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

// TODO(mheberling): is there a way to change this so that it doesn't rely only on azure service bus?
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

		// 2 Match it with the correct type of operation
		operation, err := matcher.CreateInstance(body.OperationName)
		if err != nil {
			logger.Error("Operation type doesn't exist in the matcher: " + err.Error())
			panic(err)
		}

		// 3. Init the operation with the information we have.
		operation.Init(body)

		// 4. Guard against concurrency.
		ce, err := operation.Guardconcurrency()
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + err.Error())
			logger.Error("Categorized Error calling GuardConcurrency: " + ce.Error())
			panic(err)
		}
		err = operation.Retry(ctx)
		if err != nil {
			logger.Error("Error retrying: " + err.Error())
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

// This is where we will be receiving the operation from the service bus and simply call run on them.
// func Start(ctx context.Context, r ReceiverInterface) {
// 	logger := ctxlogger.GetLogger(ctx)
//
// 	for {
// 		// 1. Receive the operation from the service bus
// 		operation, err := r.ReceiveOperation(ctx)
// 		if err != nil {
// 			logger.Error("Error calling ReceiveOperation: " + err.Error())
// 			continue
// 		}
//
// 		if operation == nil {
// 			logger.Error("Something went wrong receiving operation")
// 			continue
// 		}
//
// 		//TODO(mheberling): Here we will see if the operation has no corresponding entry in the db, or the operation Id doesn't match, etc. We need to figure out all the potential error scenarios and how to add them here while still allowing the user to define what their potential error scenarios are.
// 		// 2. Guard against concurrency.
// 		ce, err := operation.Guardconcurrency()
// 		if err != nil {
// 			logger.Error("Error calling GuardConcurrency: " + err.Error())
// 			logger.Error("Categorized Error calling GuardConcurrency: " + ce.Error())
// 			return
// 			//TODO(mheberling): Here we will have the retry, so we need to add Retry method to the APIOperation.
// 		}
//
// 		// 3. Call run on the operation
// 		operation.Run(ctx) //TODO(mheberling): Make a channel and send it through here for errors, timeout, cancellation, etc.
// 	}
// }
