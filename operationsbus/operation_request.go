package operationsbus

import (
	"context"
	"encoding/json"
	"fmt"

	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/aks-middleware/ctxlogger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	OperationName  string
	APIVersion     string
	RetryCount     int
	OperationId    string
	EntityId       string
	EntityType     string
	ExpirationDate *timestamppb.Timestamp

	// Service bus for retry
	ServiceBusConnectionString string
	ServiceBusClientUrl        string
	ServiceBusSenderQueue      string

	// HTTP
	Body       []byte
	HttpMethod string
}

func (opRequest *OperationRequest) Retry(ctx context.Context) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Retrying the long running operation.")

	var servicebusClient *sb.ServiceBus
	var err error
	if opRequest.ServiceBusConnectionString != "" {
		servicebusClient, err = sb.CreateServiceBusClientFromConnectionString(ctx, opRequest.ServiceBusConnectionString)
		if err != nil {
			logger.Error("Something went wrong creating the service bus client: " + err.Error())
			return err
		}
	} else {
		servicebusClient, err = sb.CreateServiceBusClient(ctx, opRequest.ServiceBusClientUrl)
		if err != nil {
			logger.Error("Something went wrong creating the service bus client: " + err.Error())
			return err
		}
	}

	sender, err := servicebusClient.NewServiceBusSender(ctx, opRequest.ServiceBusSenderQueue)
	if err != nil {
		logger.Error("Something went wrong creating the service bus sender: " + err.Error())
		return err
	}

	opRequest.RetryCount++
	logger.Info(fmt.Sprintf("Current retry: %d", opRequest.RetryCount))

	marshalledOperation, err := json.Marshal(opRequest)
	if err != nil {
		logger.Error("Error marshalling operation: " + err.Error())
		return err
	}

	logger.Info("Sending message to Service Bus")
	err = sender.SendMessage(ctx, []byte(marshalledOperation))
	if err != nil {
		logger.Error("Something happened: " + err.Error())
		return err
	}

	return nil
}
