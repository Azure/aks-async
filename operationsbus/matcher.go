package operationsbus

import (
	"errors"
	"reflect"
)

type EntityFactoryFunc func(string) Entity

// The matcher is utilized in order to keep track of the name and type of each operation.
// This is required because we only send the OperationRequest through the service bus,
// but we utilize the name in that struct to create an instance of the right operation
// type (e.g. LongRunning) and Run with the correct logic. The matcher can also be used
// to create the Entity based on the name of the operation by using a stored EntityFactoryFunc.
type Matcher struct {
	Types          map[string]reflect.Type
	EntityCreators map[string]EntityFactoryFunc
}

func NewMatcher() *Matcher {
	return &Matcher{
		Types:          make(map[string]reflect.Type),
		EntityCreators: make(map[string]EntityFactoryFunc),
	}
}

// Set adds a key-value pair to the map
// Ex: matcher.Register("LongRunning", &LongRunning{})
func (m *Matcher) Register(key string, value ApiOperation) {
	m.Types[key] = reflect.TypeOf(value).Elem()
}

// Set adds a key-value pair to the map
// Ex: matcher.Register("LongRunning", &LongRunning{})
func (m *Matcher) RegisterEntity(key string, value EntityFactoryFunc) {
	m.EntityCreators[key] = value
}

// Get retrieves a type from the map by its key.
func (m *Matcher) Get(key string) (reflect.Type, bool) {
	value, exists := m.Types[key]
	return value, exists
}

// This will create an empty instance of the type, with which you can then call op.Init()
// and initialize any info you need.
func (m *Matcher) CreateOperationInstance(key string) (ApiOperation, error) {
	t, exists := m.Types[key]
	if !exists {
		return nil, errors.New("The ApiOperation doesn't exist in the map: " + key)
	}

	instance := reflect.New(t).Interface().(ApiOperation)
	return instance, nil
}

// This will create an Entity using the EntityFactoryFunc with the lastOperationId by matching
// with the key passed in.
func (m *Matcher) CreateEntityInstance(key string, lastOperationId string) (Entity, error) {

	if lastOperationId == "" {
		return nil, errors.New("lastOperationId is empty!")
	}

	var entity Entity
	if f, ok := m.EntityCreators[key]; ok {
		entity = f(lastOperationId)
	} else {
		return nil, errors.New("Something went wrong getting the value of key: " + key)
	}

	if entity == nil {
		return nil, errors.New("Entity was not created successfully!")
	}

	return entity, nil
}

// Creates an instance of the operation with hooks enabled.
func (m *Matcher) CreateHookedInstace(key string, hooks []BaseOperationHooksInterface) (*HookedApiOperation, error) {
	operation, err := m.CreateOperationInstance(key)
	if err != nil {
		return nil, err
	}

	if hooks == nil {
		hooks = []BaseOperationHooksInterface{}
	}

	hOperation := &HookedApiOperation{
		Operation:      operation,
		OperationHooks: hooks,
	}

	return hOperation, nil
}
