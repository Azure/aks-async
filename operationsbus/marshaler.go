package operationsbus

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

type Marshaler interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

// The default marshaler implementation to be used by the processor.
var _ Marshaler = &ProtoMarshaler{}

type ProtoMarshaler struct{}

func (p ProtoMarshaler) Marshal(v interface{}) ([]byte, error) {
	message, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("type assertion to proto.Message failed")
	}
	return proto.Marshal(message)
}

func (p ProtoMarshaler) Unmarshal(data []byte, v interface{}) error {
	message, ok := v.(proto.Message)
	if !ok {
		return errors.New("type assertion to proto.Message failed")
	}
	return proto.Unmarshal(data, message)
}
