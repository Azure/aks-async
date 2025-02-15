package operation

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
//
// 	"log/slog"
// 	"strings"
//
// 	handlerError "github.com/Azure/aks-async/runtime/handlers/errors"
// 	operation "github.com/Azure/aks-async/runtime/operation"
// 	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
// 	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
// 	"github.com/Azure/go-shuttle/v2"
// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// )
//
// // func TestQoSErrorHandler(t *testing.T) {
// // 	RegisterFailHandler(Fail)
// // 	RunSpecs(t, "QoSErrorHandler Suite")
// // }
//
// var _ = Describe("DeadLetterQueueHandler", func() {
// 	var (
// 		// ctrl                         *gomock.Controller
// 		ctx     context.Context
// 		buf     bytes.Buffer
// 		settler shuttle.MessageSettler
// 		message *azservicebus.ReceivedMessage
// 		handler shuttle.HandlerFunc
// 		req     operation.OperationRequest
// 	)
//
// 	BeforeEach(func() {
// 		buf.Reset()
// 		// ctrl = gomock.NewController(GinkgoT())
// 		logger := slog.New(slog.NewTextHandler(&buf, nil))
// 		ctx = context.TODO()
// 		ctx = ctxlogger.WithLogger(ctx, logger)
//
// 		settler = &fakeMessageSettler{}
// 		marshalledOperation, err := json.Marshal(req)
// 		if err != nil {
// 			return
// 		}
// 		message = &azservicebus.ReceivedMessage{
// 			Body: marshalledOperation,
// 		}
// 		handler = NewQosErrorHandler(SampleErrorHandler())
// 	})
//
// 	Context("mock testing", func() {
// 		It("should handle the dead-letter queue message correctly", func() {
// 			handler(ctx, settler, message)
// 			Expect(strings.Count(buf.String(), "QoS: ")).To(Equal(3))
// 		})
// 	})
// })
//
// func SampleErrorHandler() handlerError.ErrorHandlerFunc {
// 	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
// 		return nil
// 	}
// }

// type fakeMessageSettler struct{}
//
// func (f *fakeMessageSettler) AbandonMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.AbandonMessageOptions) error {
// 	return nil
// }
// func (f *fakeMessageSettler) CompleteMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.CompleteMessageOptions) error {
// 	failureMessage := "failure_test"
// 	if message.ContentType != nil && strings.Compare(*message.ContentType, failureMessage) == 0 {
// 		return errors.New("settler error")
// 	}
// 	return nil
// }
// func (f *fakeMessageSettler) DeadLetterMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeadLetterOptions) error {
// 	return nil
// }
// func (f *fakeMessageSettler) DeferMessage(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.DeferMessageOptions) error {
// 	return nil
// }
// func (f *fakeMessageSettler) RenewMessageLock(ctx context.Context, message *azservicebus.ReceivedMessage, options *azservicebus.RenewMessageLockOptions) error {
// 	return nil
// }
