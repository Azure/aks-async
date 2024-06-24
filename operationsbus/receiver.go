package operationsbus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// The receiver interface is what the user who is implementing their own async component (what actually receives and runs the operations) will need to implement, to do with their own definitions.
type ReceiverInterface interface {
	ReceiveOperation(context.Context) (APIOperation, error)
}

type Receiver struct {
	sbusClient *azservicebus.Client //TODO(mheberling): change to an interface so that they can change to use any service bus client not just azure.
	ReceiverInterface
}

func NewReceiver(sbusClient *azservicebus.Client) *Receiver {
	receiver := &Receiver{
		sbusClient: sbusClient,
	}
	return receiver
}

// This is where we will be receiving the operation from the service bus and simply call run on them.
func Start(ctx context.Context, r ReceiverInterface) {
	logger := ctxlogger.GetLogger(ctx)

	for {
		// 1. Receive the operation from the service bus
		operation, err := r.ReceiveOperation(ctx)
		if err != nil {
			logger.Error("Error calling ReceiveOperation: " + err.Error())
			continue
		}

		if operation == nil {
			logger.Error("Something went wrong receiving operation")
			continue
		}

		//TODO(mheberling): Here we will see if the operation has no corresponding entry in the db, or the operation Id doesn't match, etc. We need to figure out all the potential error scenarios and how to add them here while still allowing the user to define what their potential error scenarios are.
		// 2. Guard against concurrency.
		ce, err := operation.Guardconcurrency()
		if err != nil {
			logger.Error("Error calling GuardConcurrency: " + err.Error())
			logger.Error("Categorized Error calling GuardConcurrency: " + ce.Error())
			return
		}

		// 3. Call run on the operation
		operation.Run(ctx) //TODO(mheberling): Make a channel and send it through here for errors
		// return
	}
}
