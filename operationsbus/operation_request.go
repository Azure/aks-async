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

	// HTTP
	Body       []byte
	HttpMethod string
}

func (opRequest *OperationRequest) Retry(ctx context.Context, sender sb.ServiceBusSender) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Retrying the long running operation.")
	logger.Info(fmt.Sprintf("Struct: %+v", opRequest))

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
