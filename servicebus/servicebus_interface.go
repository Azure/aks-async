package servicebus

import (
	"context"
)

type ServiceBusClientInterface interface {
	SendMessage(ctx context.Context, message []byte) (interface{}, error)
	ReceiveMessage(ctx context.Context, message []byte) (interface{}, error)
}
