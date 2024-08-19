package operationsbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

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

	Extension interface{}
}

func NewOperationRequest(
	operationName string,
	apiVersion string,
	operationId string,
	entityId string,
	entityType string,
	retryCount int,
	expirationDate *timestamppb.Timestamp,
	body []byte,
	httpMethod string,
	extension interface{},
) *OperationRequest {
	return &OperationRequest{
		OperationName:  operationName,
		APIVersion:     apiVersion,
		RetryCount:     retryCount,
		OperationId:    operationId,
		EntityId:       entityId,
		EntityType:     entityType,
		ExpirationDate: expirationDate,
		Body:           body,
		HttpMethod:     httpMethod,
		Extension:      extension,
	}
}

// Generalized method to retry every operation. If the operation failed or hit an error at any stage, this method will be called after the panic is handled.
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

// SetExtension sets the Extension field to a new type and value, copying data if possible
func (opRequest *OperationRequest) SetExtension(newValue interface{}) error {
	newType := reflect.TypeOf(newValue)
	if newType == nil {
		return errors.New("new value is nil")
	}

	// Create a new instance of the type
	newInstance := reflect.New(newType).Elem()

	if opRequest.Extension != nil {
		oldValue := reflect.ValueOf(opRequest.Extension)
		if oldValue.Kind() == reflect.Ptr {
			oldValue = oldValue.Elem()
		}

		if oldValue.Type().AssignableTo(newType) {
			newInstance.Set(oldValue)
		} else {
			// Handle conversion based on known types or provide a custom conversion
			data, err := json.Marshal(opRequest.Extension)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, newInstance.Addr().Interface()); err != nil {
				return err
			}
		}
	} else {
		// Initialize with zero values if Extension is nil
		newInstance.Set(reflect.Zero(newType))
	}

	// opRequest.ExtensionType = newType
	opRequest.Extension = newInstance.Interface()

	return nil
}
