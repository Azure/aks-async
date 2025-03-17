package entity

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "log/slog"
	"strings"
	"testing"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"

	oc "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQoSErrorHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EntityHooks Suite")
}

var _ = Describe("EntityHooks", func() {
	var (
		hookedOperation  *OperationStatusHook
		db               *sql.DB
		mockDb           sqlmock.Sqlmock
		ctx              context.Context
		req              operation.OperationRequest
		query            string
		sampleError      error
		op               *sampleOperation.SampleOperation
		categorizedError *entity.CategorizedError
		buf              bytes.Buffer
	)

	BeforeEach(func() {
		ctx = context.Background()
		buf.Reset()
		logger := log.New(log.NewTextHandler(&buf, nil))
		ctx = ctxlogger.WithLogger(ctx, logger)
		req = operation.OperationRequest{
			EntityId:   "test_entity_id",
			EntityType: "test_entity_type",
		}
		db, mockDb, _ = sqlmock.New()
		hookedOperation = &OperationStatusHook{
			dbClient:        db,
			EntityTableName: "test_table",
		}
		op = &sampleOperation.SampleOperation{}
		_, err := op.InitOperation(ctx, req)
		Expect(err).To(BeNil())

		errorMessage := "Error message"
		sampleError = errors.New(errorMessage)
		categorizedError = &entity.CategorizedError{
			Err: sampleError,
		}
		query = fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND EXISTS \(SELECT 1 FROM %s WHERE entity_id = @p2 AND entity_type = @p3\)`, hookedOperation.EntityTableName, hookedOperation.EntityTableName)
	})

	AfterEach(func() {
		err := mockDb.ExpectationsWereMet()
		Expect(err).To(BeNil())
	})

	Describe("BeforeInitOperation", func() {
		Context("when there is no error", func() {
			It("should update entity database with IN_PROGRESS status", func() {
				inProgressOperationStatus := oc.Status_IN_PROGRESS.String()
				mockDb.ExpectExec(query).WithArgs(inProgressOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewResult(1, 1))

				err := hookedOperation.BeforeInitOperation(ctx, req)
				Expect(err).To(BeNil())
			})
		})
		Context("when there is a query error", func() {
			It("should return an error", func() {
				inProgressOperationStatus := oc.Status_IN_PROGRESS.String()
				mockDb.ExpectExec(query).WithArgs(inProgressOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewErrorResult(sampleError))

				err := hookedOperation.BeforeInitOperation(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "Error updating the entity database of entity with id: "+req.EntityId+" and type: "+req.EntityType+" to IN_PROGRESS status: ")).To(Equal(1))
			})
		})
	})

	Describe("AfterInitOperation", func() {
		Context("when Init returned an error", func() {
			It("should update entity database with PENDING status", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewResult(1, 1))

				err := hookedOperation.AfterInitOperation(ctx, op, req, sampleError)
				Expect(err).To(BeNil())
			})
			It("should return en error if query fails", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewErrorResult(sampleError))

				err := hookedOperation.AfterInitOperation(ctx, op, req, sampleError)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "Error updating the entity database of entity with id: "+req.EntityId+" and type: "+req.EntityType+" to PENDING status: ")).To(Equal(1))
			})
		})
		Context("when Init didn't return an error", func() {
			It("should do nothing", func() {
				err := hookedOperation.AfterInitOperation(ctx, op, req, nil)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("BeforeGuardConcurrency", func() {
		It("should do nothing", func() {
			err := hookedOperation.BeforeGuardConcurrency(ctx, op, nil)
			Expect(err).To(BeNil())
		})
	})

	Describe("AfterGuardConcurrency", func() {
		Context("when there is a categorized error", func() {
			It("should update entity database with PENDING status", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewResult(1, 1))

				err := hookedOperation.AfterGuardConcurrency(ctx, op, categorizedError)
				Expect(err).To(BeNil())
			})
			It("should fail if query fails", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewErrorResult(sampleError))

				err := hookedOperation.AfterGuardConcurrency(ctx, op, categorizedError)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "Error updating the entity database of entity with id: "+req.EntityId+" and type: "+req.EntityType+" to PENDING status: ")).To(Equal(1))
			})
		})
		Context("when there is no categorized error", func() {
			It("should do nothing", func() {
				err := hookedOperation.AfterGuardConcurrency(ctx, op, nil)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("BeforeRun", func() {
		It("should do nothing", func() {
			err := hookedOperation.BeforeRun(ctx, op)
			Expect(err).To(BeNil())
		})
	})

	Describe("AfterRun", func() {
		Context("when Run returned an error", func() {
			It("should update entity database with PENDING status", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewResult(1, 1))

				err := hookedOperation.AfterRun(ctx, op, sampleError)
				Expect(err).To(BeNil())
			})
			It("should return an error if query fails", func() {
				pendingOperationStatus := oc.Status_PENDING.String()
				mockDb.ExpectExec(query).WithArgs(pendingOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewErrorResult(sampleError))

				err := hookedOperation.AfterRun(ctx, op, sampleError)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "Error updating the entity database of entity with id: "+req.EntityId+" and type: "+req.EntityType+" to PENDING status: ")).To(Equal(1))
			})
		})

		Context("when there is no error", func() {
			It("should update entity database with COMPLETED status", func() {
				completedOperationStatus := oc.Status_COMPLETED.String()
				mockDb.ExpectExec(query).WithArgs(completedOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewResult(1, 1))

				err := hookedOperation.AfterRun(ctx, op, nil)
				Expect(err).To(BeNil())
			})
			It("should return an error if query fails", func() {
				completedOperationStatus := oc.Status_COMPLETED.String()
				mockDb.ExpectExec(query).WithArgs(completedOperationStatus, req.EntityId, req.EntityType).WillReturnResult(sqlmock.NewErrorResult(sampleError))

				err := hookedOperation.AfterRun(ctx, op, nil)
				Expect(err).ToNot(BeNil())
				Expect(strings.Count(err.Error(), "Error updating the entity database of entity with id: "+req.EntityId+" and type: "+req.EntityType+" to COMPLETED status: ")).To(Equal(1))
			})
		})
	})
})
