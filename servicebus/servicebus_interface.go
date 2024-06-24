package servicebus

import (
	"context"
)

type ServiceBusClientInterface interface {
	NewServiceBusClient(ctx context.Context, connectionString string, senderQueueName string, receiverQueueName string) (ServiceBusClientInterface, error)
	SendMessage(ctx context.Context, message []byte) (interface{}, error) //TODO(mheberling): Here we can start returning the httpResponse with response code maybe?
	ReceiveMessage(ctx context.Context, message []byte) (interface{}, error)
}
