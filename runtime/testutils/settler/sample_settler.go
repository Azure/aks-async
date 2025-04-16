package settler

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

type SampleMessageSettler struct{}

func (f *SampleMessageSettler) AbandonMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.AbandonMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *SampleMessageSettler) CompleteMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *SampleMessageSettler) DeadLetterMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *SampleMessageSettler) DeferMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeferMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *SampleMessageSettler) RenewMessageLock(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.RenewMessageLockOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
