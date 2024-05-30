package servicebus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

type ServiceBus struct {
	Client   *azservicebus.Client
	Receiver *azservicebus.Receiver
	Sender   *azservicebus.Sender
}

func GetClient(connectionString string) (*azservicebus.Client, error) {
	// logger := ctxlogger.GetLogger(ctx)
	// logger.Info("Send message!")

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	// client, err := azservicebus.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewServiceBusClient(ctx context.Context, connectionString string, senderQueueName string, receiverQueueName string) (*ServiceBus, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("New Service Bus client!")

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	// client, err := azservicebus.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, err
	}

	var sender *azservicebus.Sender
	if receiverQueueName != "" {
		sender, err = client.NewSender(senderQueueName, nil)
		if err != nil {
			logger.Info("Error creating sender!")
			return nil, err
		}
	}

	var receiver *azservicebus.Receiver
	if receiverQueueName != "" {
		receiver, err = client.NewReceiverForQueue(receiverQueueName, nil)
		if err != nil {
			logger.Info("Error creating receiver!")
			return nil, err
		}
	}

	return &ServiceBus{
		Client:   client,
		Receiver: receiver,
		Sender:   sender,
	}, nil
}

// We send and receive with []byte, because it's generic enough if someone wants to marshall it through a different method.
func (sb *ServiceBus) SendMessage(ctx context.Context, message []byte) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Send message!")

	azMessage := &azservicebus.Message{
		Body: message,
	}

	err := sb.Sender.SendMessage(ctx, azMessage, nil)
	if err != nil {
		logger.Info("Error sending message!")
		return err
	}

	return nil
}

// Ditto the above
func (sb *ServiceBus) ReceiveMessage(ctx context.Context) ([]byte, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Receive message!")
	messages, err := sb.Receiver.ReceiveMessages(ctx, 1, nil)
	if err != nil {
		logger.Info("Error receiving message!")
		return nil, err
	}

	var body []byte
	for _, message := range messages {
		body = message.Body
		logger.Info("%s\n" + string(body))

		err = sb.Receiver.CompleteMessage(ctx, message, nil)
		if err != nil {
			logger.Info("Error completing message!")
			return nil, err
		}
	}

	return body, nil
}
