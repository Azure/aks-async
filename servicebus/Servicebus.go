package servicebus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// type ServiceBus struct {
// 	Client   *azservicebus.Client
// 	Receiver *azservicebus.Receiver //TODO(mheberling): These two are not needed, we just need the client and the methods to create the receiver and sender.
// 	Sender   *azservicebus.Sender
// 	//TODO(mheberling): Add a marshaller here? To receive and send the messages.
// }

type ServiceBus struct {
	//TODO(mheberling): In the future, we can add more types of service bus here.
	Client *azservicebus.Client
}

type ServiceBusReceiver struct {
	Receiver *azservicebus.Receiver
}

type ServiceBusSender struct {
	Sender *azservicebus.Sender
}

func CreateServiceBusClient(ctx context.Context) (*ServiceBus, error) {

	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating Service Bus!")

	tokenCredential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("Error getting tokne credential")
		return nil, err
	}

	client, err := azservicebus.NewClient("heberling.servicebus.windows.net", tokenCredential, nil)
	if err != nil {
		logger.Error("Error getting client.")
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}

func CreateServiceBusClientFromConnectionString(ctx context.Context, connectionString string) (*ServiceBus, error) {

	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating Service Bus from Connection String!")

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		logger.Error("Error getting client.")
		return nil, err
	}

	servicebus := &ServiceBus{
		Client: client,
	}

	return servicebus, nil
}
func (sb *ServiceBus) NewServiceBusReceiver(ctx context.Context, topicOrQueue string) (*ServiceBusReceiver, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus receiver.")

	receiver, err := sb.Client.NewReceiverForQueue("servicehub", nil)
	if err != nil {
		logger.Error("Error getting receiver.")
		return nil, err
	}

	serviceBusReceiver := &ServiceBusReceiver{
		Receiver: receiver,
	}

	return serviceBusReceiver, nil
}

func (sb *ServiceBus) NewServiceBusSender(ctx context.Context, queue string) (*ServiceBusSender, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating new service bus sender.")

	sender, err := sb.Client.NewSender("servicehub", nil)
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

	fakeMessage := &azservicebus.Message{
		Body: message,
	}

	err := s.Sender.SendMessage(ctx, fakeMessage, nil)
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

// func GetClient(connectionString string) (*azservicebus.Client, error) {
// 	// logger := ctxlogger.GetLogger(ctx)
// 	// logger.Info("Send message!")
//
// 	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
// 	// client, err := azservicebus.NewClient(namespace, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return client, nil
// }
//
// func NewServiceBusClient(ctx context.Context, connectionString string, senderQueueName string, receiverQueueName string) (*ServiceBus, error) {
// 	logger := ctxlogger.GetLogger(ctx)
// 	logger.Info("New Service Bus client!")
//
// 	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
// 	// client, err := azservicebus.NewClient(namespace, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var sender *azservicebus.Sender
// 	if receiverQueueName != "" {
// 		sender, err = client.NewSender(senderQueueName, nil)
// 		if err != nil {
// 			logger.Info("Error creating sender!")
// 			return nil, err
// 		}
// 	}
//
// 	var receiver *azservicebus.Receiver
// 	if receiverQueueName != "" {
// 		receiver, err = client.NewReceiverForQueue(receiverQueueName, nil)
// 		if err != nil {
// 			logger.Info("Error creating receiver!")
// 			return nil, err
// 		}
// 	}
//
// 	return &ServiceBus{
// 		Client:   client,
// 		Receiver: receiver,
// 		Sender:   sender,
// 	}, nil
// }

// We send and receive with []byte, because it's generic enough if someone wants to marshall it through a different method.
// func (sb *ServiceBus) SendMessage(ctx context.Context, message []byte) error {
// 	logger := ctxlogger.GetLogger(ctx)
// 	logger.Info("Send message!")
//
// 	azMessage := &azservicebus.Message{
// 		Body: message,
// 	}
//
// 	err := sb.Sender.SendMessage(ctx, azMessage, nil)
// 	if err != nil {
// 		logger.Info("Error sending message!")
// 		return err
// 	}
//
// 	return nil
// }
//
// // Ditto the above
// func (sb *ServiceBus) ReceiveMessage(ctx context.Context) ([]byte, error) {
// 	logger := ctxlogger.GetLogger(ctx)
// 	logger.Info("Receive message!")
// 	messages, err := sb.Receiver.ReceiveMessages(ctx, 1, nil)
// 	if err != nil {
// 		logger.Info("Error receiving message!")
// 		return nil, err
// 	}
//
// 	var body []byte
// 	for _, message := range messages {
// 		body = message.Body
// 		logger.Info("%s\n" + string(body))
//
// 		err = sb.Receiver.CompleteMessage(ctx, message, nil)
// 		if err != nil {
// 			logger.Info("Error completing message!")
// 			return nil, err
// 		}
// 	}
//
// 	return body, nil
// }
