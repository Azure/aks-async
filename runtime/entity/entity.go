package entity

// This is the entity that will be grabbed from the database by each implementer. They will have to use this GetLatestOperationId() method when guarding for concurrency.
type Entity interface {
	GetLatestOperationID() string
}
