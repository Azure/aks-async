package servicebus

import (
	"context"
)

type ServiceBusClientInterface interface {
	NewServiceBusReceiver(ctx context.Context, topicOrQueue string) (ReceiverInterface, error)
	NewServiceBusSender(ctx context.Context, queue string) (SenderInterface, error)
}

type SenderInterface interface {
	SendMessage(ctx context.Context, message []byte) error
}

type ReceiverInterface interface {
	ReceiveMessage(ctx context.Context) ([]byte, error)
}
