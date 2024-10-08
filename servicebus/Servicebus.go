package servicebus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

type ServiceBus struct {
	// In the future, we can add more types of service bus here.
	Client *azservicebus.Client
}

type ServiceBusReceiver struct {
	Receiver *azservicebus.Receiver
}

type ServiceBusSender struct {
	Sender *azservicebus.Sender
}

func CreateServiceBusClient(ctx context.Context, clientUrl string, credential azcore.TokenCredential, options *azservicebus.ClientOptions) (*ServiceBus, error) {

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
		logger.Error("Error getting client.")
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}

func CreateServiceBusClientFromConnectionString(ctx context.Context, connectionString string, options *azservicebus.ClientOptions) (*ServiceBus, error) {

	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating Service Bus from Connection String!")

	client, err := azservicebus.NewClientFromConnectionString(connectionString, options)
	if err != nil {
		logger.Error("Error getting client.")
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}
func (sb *ServiceBus) NewServiceBusReceiver(ctx context.Context, topicOrQueue string, options *azservicebus.ReceiverOptions) (*ServiceBusReceiver, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus receiver.")

	receiver, err := sb.Client.NewReceiverForQueue(topicOrQueue, options)
	if err != nil {
		logger.Error("Error getting receiver.")
		return nil, err
	}

	serviceBusReceiver := &ServiceBusReceiver{
		Receiver: receiver,
	}

	return serviceBusReceiver, nil
}

func (sb *ServiceBus) NewServiceBusSender(ctx context.Context, queue string, options *azservicebus.NewSenderOptions) (*ServiceBusSender, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus sender.")

	sender, err := sb.Client.NewSender(queue, options)
	if err != nil {
		logger.Error("Error getting the sender")
		return nil, err
	}

	serviceBusSender := &ServiceBusSender{
		Sender: sender,
	}

	return serviceBusSender, nil
}

func (s *ServiceBusSender) SendMessage(ctx context.Context, message []byte) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Sending message through service bus sender.")

	packagedMessage := &azservicebus.Message{
		Body: message,
	}

	err := s.Sender.SendMessage(ctx, packagedMessage, nil)
	if err != nil {
		logger.Error("Error Sending message")
		return err
	}

	logger.Info("Message sent successfully!")
	return nil
}

// TODO(mheberling): Don't think this is necessary.
func (r *ServiceBusReceiver) ReceiveMessage(ctx context.Context) ([]byte, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus sender.")

	messages, err := r.Receiver.ReceiveMessages(ctx, 1, nil)
	if err != nil {
		logger.Info("Error receiving message!")
		return nil, err
	}

	var body []byte
	for _, message := range messages {
		body = message.Body
		logger.Info("%s\n" + string(body))

		err = r.Receiver.CompleteMessage(ctx, message, nil)
		if err != nil {
			logger.Info("Error completing message!")
			return nil, err
		}
	}

	return body, nil
}
