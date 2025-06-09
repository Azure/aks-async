package convert

import (
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

func ConvertToReceivedMessage(msg *azservicebus.Message) *azservicebus.ReceivedMessage {
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
