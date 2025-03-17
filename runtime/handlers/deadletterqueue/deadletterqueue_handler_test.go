package deadletterqueue

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	log "log/slog"
	"strings"
	"testing"

	oc "github.com/Azure/OperationContainer/api/v1"
	ocMock "github.com/Azure/OperationContainer/api/v1/mock"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-async/runtime/testutils/settler"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeadLetterQueueHandler Suite")
}

var _ = Describe("DeadLetterQueueHandler", func() {
	var (
		ctrl                         *gomock.Controller
		ctx                          context.Context
		buf                          bytes.Buffer
		operationContainerClient     *ocMock.MockOperationContainerClient
		db                           *sql.DB
		mockDb                       sqlmock.Sqlmock
		sampleSettler                shuttle.MessageSettler
		entityTableName              string
		message                      *azservicebus.ReceivedMessage
		handler                      shuttle.HandlerFunc
		query                        string
		req                          operation.OperationRequest
		failedOperationStatus        oc.Status
		updateOperationStatusRequest *oc.UpdateOperationStatusRequest
	)

	BeforeEach(func() {
		buf.Reset()
		ctrl = gomock.NewController(GinkgoT())
		logger := log.New(log.NewTextHandler(&buf, nil))
		ctx = context.TODO()
		ctx = ctxlogger.WithLogger(ctx, logger)
		operationContainerClient = ocMock.NewMockOperationContainerClient(ctrl)
		db, mockDb, _ = sqlmock.New()

		sampleSettler = &settler.SampleMessageSettler{}
		entityTableName = "test_entity_table_name"
		failedOperationStatus = oc.Status_FAILED
		req = operation.OperationRequest{
			EntityId:    "test_entity_id",
			EntityType:  "test_entity_type",
			OperationId: "test_operation_id",
		}
		marshalledOperation, err := json.Marshal(req)
		if err != nil {
			return
		}
		message = &azservicebus.ReceivedMessage{
			Body: marshalledOperation,
		}
		updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
			OperationId: req.OperationId,
			Status:      failedOperationStatus,
		}
		query = fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND entity_type = @p3 AND last_operation_id = @p4\;`, entityTableName)
		handler = NewDeadLetterQueueHandler(entityTableName, operationContainerClient, db)
	})

	Context("mock testing", func() {
		It("should handle the dead-letter queue message correctly", func() {
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, nil)
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Successfully set the operation")).To(Equal(1))
		})
		It("should throw an error if unmarshal fails", func() {
			message = &azservicebus.ReceivedMessage{
				Body: []byte(`invalid json`),
			}
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "Error unmarshalling message")).To(Equal(1))
		})
		It("should throw error if query fails", func() {
			err := errors.New("Sample error")
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewErrorResult(err))
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Error in entity table query")).To(Equal(1))
		})
		It("should throw error if OperationContainerClient fails", func() {
			err := errors.New("Sample error")
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, err)
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "DeadLetterQueueHandler: Error setting operation")).To(Equal(1))
		})
		It("should throw error if settler fails", func() {
			contentType := "failure_test"
			message.ContentType = &contentType
			mockDb.ExpectExec(query).WithArgs(failedOperationStatus.String(), req.EntityId, req.EntityType, req.OperationId).WillReturnResult(sqlmock.NewResult(1, 1))
			operationContainerClient.EXPECT().UpdateOperationStatus(gomock.Any(), updateOperationStatusRequest).Return(nil, nil)
			handler(ctx, sampleSettler, message)
			Expect(strings.Count(buf.String(), "Unable to settle message")).To(Equal(1))
			Expect(strings.Count(buf.String(), "settler error")).To(Equal(1))
		})
	})
})
