package entity

// This is the entity that will be grabbed from the database by each implementer. They will have to use this GetLatestOperationId() method when guarding for concurrency.
type Entity interface {
	GetLatestOperationID() string
}

// The EntityFactoryFunc is utilized as an interface for a function to create
// entity.Entity types.
type EntityFactoryFunc func(string) (Entity, error)
