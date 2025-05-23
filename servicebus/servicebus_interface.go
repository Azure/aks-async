package servicebus

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

type ServiceBusClientInterface interface {
	NewServiceBusReceiver(ctx context.Context, topicOrQueue string, options *azservicebus.ReceiverOptions) (ReceiverInterface, error)
	NewServiceBusSender(ctx context.Context, queue string, options *azservicebus.NewSenderOptions) (SenderInterface, error)
}

type SenderInterface interface {
	SendMessage(ctx context.Context, message *azservicebus.Message) error
	GetAzureSender() (*azservicebus.Sender, error)
}

type ReceiverInterface interface {
	ReceiveMessage(ctx context.Context) ([]*azservicebus.ReceivedMessage, error)
	GetAzureReceiver() (*azservicebus.Receiver, error)
}
