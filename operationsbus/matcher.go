package operationsbus

import (
	"errors"
	"reflect"
)

// The matcher is utilized in order to keep track of the name and type of each operation. This is required because we only send the OperationRequest through the service bus, but we utilize the name shown in that struct in order to create an instance of the right operation type (e.g. LongRunning) and Run with the correct logic.
type Matcher struct {
	Types map[string]reflect.Type
}

func NewMatcher() *Matcher {
	return &Matcher{
		Types: make(map[string]reflect.Type),
	}
}

// Set adds a key-value pair to the map
// Ex: matcher.Register("LongRunning", &LongRunning{})
func (m *Matcher) Register(key string, value ApiOperation) {
	m.Types[key] = reflect.TypeOf(value).Elem()
}

// Set adds a key-value pair to the map
// Ex: matcher.Register("LongRunning", &LongRunning{})
func (m *Matcher) RegisterEntity(key string, value Entity) {
	m.Types[key] = reflect.TypeOf(value).Elem()
}

// Get retrieves a value from the map by its key
func (m *Matcher) Get(key string) (reflect.Type, bool) {
	value, exists := m.Types[key]
	return value, exists
}

// This will create an empty instance of the type, with which you can then call op.Init() and initialize any info you need.
func (m *Matcher) CreateOperationInstance(key string) (ApiOperation, error) {
	t, exists := m.Types[key]
	if !exists {
		return nil, errors.New("The ApiOperation doesn't exist in the map: " + key)
	}

	instance := reflect.New(t).Interface().(ApiOperation)
	return instance, nil
}

// This will create an empty instance of the type, with which you can then call op.Init() and initialize any info you need.
func (m *Matcher) CreateEntityInstance(key string) (Entity, error) {
	t, exists := m.Types[key]
	if !exists {
		return nil, errors.New("The ApiOperation doesn't exist in the map: " + key)
	}

	instance := reflect.New(t).Interface().(Entity)
	return instance, nil
}

func (m *Matcher) CreateHookedInstace(key string, hooks []BaseOperationHooksInterface) (*HookedApiOperation, error) {
	operation, err := m.CreateOperationInstance(key)
	if err != nil {
		return nil, err
	}

	if hooks == nil {
		hooks = []BaseOperationHooksInterface{}
	}

	hOperation := &HookedApiOperation{
		Operation:      &operation,
		OperationHooks: hooks,
	}

	return hOperation, nil
}
