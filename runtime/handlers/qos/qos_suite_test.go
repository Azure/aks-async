package qos

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQoSHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QoSHandler Suite")
}

func SampleHandler() shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
	}
}

type fakeMessageSettler struct{}

func (f *fakeMessageSettler) AbandonMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.AbandonMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) CompleteMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
	failureMessage := "failure_test"
	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
		return errors.New("settler error")
	}
	return nil
}
func (f *fakeMessageSettler) DeadLetterMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) error {
	return nil
}
func (f *fakeMessageSettler) DeferMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeferMessageOptions) error {
	return nil
}
func (f *fakeMessageSettler) RenewMessageLock(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.RenewMessageLockOptions) error {
	return nil
}
