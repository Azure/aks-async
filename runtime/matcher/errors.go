package matcher

import (
	"fmt"
)

type OperationKeyLookupError struct {
	Key string
}

func (e *OperationKeyLookupError) Error() string {
	return fmt.Sprintf("The key %s doesn't exist in the map.", e.Key)
}

type EntityCreationKeyLookupError struct {
	Key         string
	OperationId string
}

func (e *EntityCreationKeyLookupError) Error() string {
	return fmt.Sprintf("The key %s doesn't exist in the map for operation %s.", e.Key, e.OperationId)
}

type EmptyOperationId struct{}

func (e *EmptyOperationId) Error() string {
	return fmt.Sprintf("No OperationId provided.")
}

type EntityCreationError struct{}

func (e *EntityCreationError) Error() string {
	return fmt.Sprintf("Entity is nil after creation.")
}
