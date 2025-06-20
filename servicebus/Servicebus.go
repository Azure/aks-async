package servicebus

import (
	"context"
	"errors"

	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// TODO(mheberling): Find how to test without our interface.
type ServiceBus struct {
	Client *azservicebus.Client
}

type ServiceBusReceiver struct {
	Receiver *azservicebus.Receiver
}

type ServiceBusSender struct {
	Sender *azservicebus.Sender
}

func CreateServiceBusClient(ctx context.Context, clientUrl string, credential azcore.TokenCredential, options *azservicebus.ClientOptions) (ServiceBusClientInterface, error) {

	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating Service Bus!")

	if credential == nil {
		var err error
		credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			logger.Error("Error getting token credential")
			return nil, err
		}
	}

	client, err := azservicebus.NewClient(clientUrl, credential, options)
	if err != nil {
		logger.Error("Error getting service bus client: " + err.Error())
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}

func CreateServiceBusClientFromConnectionString(ctx context.Context, connectionString string, options *azservicebus.ClientOptions) (ServiceBusClientInterface, error) {

	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating Service Bus from Connection String!")

	client, err := azservicebus.NewClientFromConnectionString(connectionString, options)
	if err != nil {
		logger.Error("Error getting service bus client: " + err.Error())
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}
func (sb *ServiceBus) NewServiceBusReceiver(ctx context.Context, topicOrQueue string, options *azservicebus.ReceiverOptions) (ReceiverInterface, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus receiver.")

	receiver, err := sb.Client.NewReceiverForQueue(topicOrQueue, options)
	if err != nil {
		logger.Error("Error getting service bus receiver: " + err.Error())
		return nil, err
	}

	serviceBusReceiver := &ServiceBusReceiver{
		Receiver: receiver,
	}

	return serviceBusReceiver, nil
}

func (sb *ServiceBus) NewServiceBusSender(ctx context.Context, queue string, options *azservicebus.NewSenderOptions) (SenderInterface, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus sender.")

	sender, err := sb.Client.NewSender(queue, options)
	if err != nil {
		logger.Error("Error getting the service bus sender: " + err.Error())
		return nil, err
	}

	serviceBusSender := &ServiceBusSender{
		Sender: sender,
	}

	return serviceBusSender, nil
}

func (s *ServiceBusSender) SendMessage(ctx context.Context, message *azservicebus.Message) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Sending message through service bus sender.")

	err := s.Sender.SendMessage(ctx, message, nil)
	if err != nil {
		logger.Error("Error Sending message: " + err.Error())
		return err
	}

	logger.Info("Message sent successfully!")
	return nil
}

func (s *ServiceBusSender) GetAzureSender() (*azservicebus.Sender, error) {
	if s.Sender != nil {
		return s.Sender, nil
	} else {
		return nil, errors.New("No Sender was found.")
	}
}

func (s *ServiceBusReceiver) GetAzureReceiver() (*azservicebus.Receiver, error) {
	if s.Receiver != nil {
		return s.Receiver, nil
	} else {
		return nil, errors.New("No Receiver was found.")
	}
}

func (r *ServiceBusReceiver) ReceiveMessage(ctx context.Context, maxMessages int, options *azservicebus.ReceiveMessagesOptions) ([]*azservicebus.ReceivedMessage, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Receiving message")

	messages, err := r.Receiver.ReceiveMessages(ctx, maxMessages, options)
	if err != nil {
		logger.Info("Error receiving message: " + err.Error())
		return nil, err
	}

	return messages, nil
}
