package matcher

import (
	"context"
	"reflect"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/operation"
)

// The matcher is utilized in order to keep track of the name and type of each operation.
// This is required because we only send the OperationRequest through the service bus,
// but we utilize the name in that struct to create an instance of the right operation
// type (e.g. LongRunning) and Run with the correct logic. The matcher can also be used
// to create the Entity based on the name of the operation by using a stored EntityFactoryFunc.
type Matcher struct {
	Types          map[string]reflect.Type
	EntityCreators map[string]entity.EntityFactoryFunc
}

func NewMatcher() *Matcher {
	return &Matcher{
		Types:          make(map[string]reflect.Type),
		EntityCreators: make(map[string]entity.EntityFactoryFunc),
	}
}

// Set adds a key-value pair to the map
// Ex: matcher.Register("LongRunning", &LongRunning{})
func (m *Matcher) Register(ctx context.Context, key string, value operation.ApiOperation) {
	m.Types[key] = reflect.TypeOf(value).Elem()
}

// Set adds a key-value pair to the map
// Ex: matcher.RegisterEntity("LongRunning", longRunningOperation.CreateLroEntityFunc)
func (m *Matcher) RegisterEntity(ctx context.Context, key string, value entity.EntityFactoryFunc) {
	m.EntityCreators[key] = value
}

// Get retrieves a type from the map by its key.
func (m *Matcher) Get(ctx context.Context, key string) (reflect.Type, bool) {
	value, exists := m.Types[key]
	return value, exists
}

// This will create an empty instance of the type, with which you can then call op.Init()
// and initialize any info you need.
func (m *Matcher) CreateOperationInstance(ctx context.Context, key string) (operation.ApiOperation, error) {
	t, exists := m.Types[key]
	if !exists {
		return nil, &OperationKeyLookupError{Key: key}
	}

	instance := reflect.New(t).Interface().(operation.ApiOperation)
	return instance, nil
}

// This will create an Entity using the EntityFactoryFunc with the lastOperationId by matching
// with the key passed in.
func (m *Matcher) CreateEntityInstance(ctx context.Context, key string, lastOperationId string) (entity.Entity, error) {

	if lastOperationId == "" {
		return nil, &EmptyOperationId{}
	}

	var entity entity.Entity
	var err error
	if f, ok := m.EntityCreators[key]; ok {
		entity, err = f(lastOperationId)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &EntityCreationKeyLookupError{Key: key}
	}

	if entity == nil {
		return nil, &EntityCreationError{}
	}

	return entity, nil
}

// Creates an instance of the operation with hooks enabled.
func (m *Matcher) CreateHookedInstance(ctx context.Context, key string, hookList []hooks.BaseOperationHooksInterface) (*hooks.HookedApiOperation, error) {
	operationInstance, err := m.CreateOperationInstance(ctx, key)
	if err != nil {
		return nil, err
	}

	if hookList == nil {
		hookList = []hooks.BaseOperationHooksInterface{}
	}

	hOperation := &hooks.HookedApiOperation{
		OperationInstance: operationInstance,
		OperationHooks:    hookList,
	}

	return hOperation, nil
}
