package operationsbus

import (
	"encoding/json"
	"errors"
	"reflect"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// All the fields that the operations might need. This struct will be part of every operation.
type OperationRequest struct {
	OperationName       string                 // Name of the operation being processed. Used to match the ApiOperation with the right implementation.
	APIVersion          string                 // Specifies the version of the API the operation is associated with, ensuring compatibility.
	RetryCount          int                    // Tracks the number of retries of the operation to prevent infinite looping or special logic around retries.
	OperationId         string                 // A unique identifier for the operation.
	EntityId            string                 // A unique identifier for the entity (resource) the operation is acting on, used with EntityType to ensure we have selected the right entity.
	EntityType          string                 // Specified the type of entity the operation is acting on, used with EntityId to ensure we have selected the right Entity.
	ExpirationTimestamp *timestamppb.Timestamp // Defines when the operation should expire and prevent execution, should it have passed this date.

	// HTTP
	Body       []byte // Contains request payload or data needed for the operation in HTTP operations.
	HttpMethod string // Indicating the HTTP method if the operation requires HTTP-based communication.

	Extension interface{} // An optional and flexible field to add any data the user may require.
}

func NewOperationRequest(
	operationName string,
	apiVersion string,
	operationId string,
	entityId string,
	entityType string,
	retryCount int,
	expirationTimestamp *timestamppb.Timestamp,
	body []byte,
	httpMethod string,
	extension interface{},
) *OperationRequest {
	return &OperationRequest{
		OperationName:       operationName,
		APIVersion:          apiVersion,
		RetryCount:          retryCount,
		OperationId:         operationId,
		EntityId:            entityId,
		EntityType:          entityType,
		ExpirationTimestamp: expirationTimestamp,
		Body:                body,
		HttpMethod:          httpMethod,
		Extension:           extension,
	}
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
		return errors.New("No extension set for operation.")
	}

	// opRequest.ExtensionType = newType
	opRequest.Extension = newInstance.Interface()

	return nil
}
