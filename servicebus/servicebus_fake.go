package servicebus

import (
	"context"
	"errors"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// In the future, support for multiple queues of messages might also be required.
type FakeServiceBusClient struct {
	messages []*azservicebus.Message
	mu       sync.Mutex
}

func NewFakeServiceBusClient() *FakeServiceBusClient {
	return &FakeServiceBusClient{
		messages: make([]*azservicebus.Message, 0),
	}
}

func (f *FakeServiceBusClient) NewServiceBusReceiver(_ context.Context, _ string, _ *azservicebus.ReceiverOptions) (ReceiverInterface, error) {
	return &FakeReceiver{
		client: f,
	}, nil
}

func (f *FakeServiceBusClient) NewServiceBusSender(_ context.Context, _ string, _ *azservicebus.NewSenderOptions) (SenderInterface, error) {
	return &FakeSender{
		client: f,
	}, nil
}

type FakeSender struct {
	client *FakeServiceBusClient
}

func (s *FakeSender) SendMessage(_ context.Context, message *azservicebus.Message) error {
	s.client.mu.Lock()
	defer s.client.mu.Unlock()

	// Append message to the slice (acting as a queue).
	s.client.messages = append(s.client.messages, message)
	return nil
}

func (s *FakeSender) GetAzureSender() (*azservicebus.Sender, error) {
	return nil, nil
}

type FakeReceiver struct {
	client *FakeServiceBusClient
}

func (r *FakeReceiver) ReceiveMessage(_ context.Context) ([]*azservicebus.ReceivedMessage, error) {
	r.client.mu.Lock()
	defer r.client.mu.Unlock()

	if len(r.client.messages) == 0 {
		return nil, errors.New("No messages available.")
	}

	message := r.client.messages[0]
	r.client.messages = r.client.messages[1:]

	receivedMessage := convertToReceivedMessage(message)
	return []*azservicebus.ReceivedMessage{receivedMessage}, nil
}

func (s *FakeReceiver) GetAzureReceiver() (*azservicebus.Receiver, error) {
	return nil, nil
}

func convertToReceivedMessage(msg *azservicebus.Message) *azservicebus.ReceivedMessage {
	var messageID string
	if msg.MessageID != nil {
		messageID = *msg.MessageID
	}

	return &azservicebus.ReceivedMessage{
		ApplicationProperties: msg.ApplicationProperties,
		Body:                  msg.Body,
		ContentType:           msg.ContentType,
		CorrelationID:         msg.CorrelationID,
		MessageID:             messageID,
		PartitionKey:          msg.PartitionKey,
		ReplyTo:               msg.ReplyTo,
		ReplyToSessionID:      msg.ReplyToSessionID,
		ScheduledEnqueueTime:  msg.ScheduledEnqueueTime,
		SessionID:             msg.SessionID,
		Subject:               msg.Subject,
		TimeToLive:            msg.TimeToLive,
		To:                    msg.To,

		// The rest of the fields like LockToken, SequenceNumber, etc., are not present in Message
		// and would need to be mocked or left as zero values if needed.
	}
}
