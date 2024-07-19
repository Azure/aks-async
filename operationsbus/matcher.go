package operationsbus

import (
	"errors"
	"reflect"
)

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
func (m *Matcher) Set(key string, value APIOperation) {
	m.Types[key] = reflect.TypeOf(value).Elem()
}

// Get retrieves a value from the map by its key
func (m *Matcher) Get(key string) (reflect.Type, bool) {
	value, exists := m.Types[key]
	return value, exists
}

// This will create an empty instance of the type, with which you can then call op.Init() and initialize any info you need.
func (m *Matcher) CreateInstance(key string) (APIOperation, error) {
	t, exists := m.Types[key]
	if !exists {
		return nil, errors.New("The APIOperation doesn't exist in the map!")
	}

	instance := reflect.New(t).Interface().(APIOperation)
	return instance, nil
}
