package deadletterqueue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	oc "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"

	"github.com/Azure/aks-async/database"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// This handler will handle updating the entity and operation store databases to a terminal
// state after being sent to the dead-letter queue.
func NewDeadLetterQueueHandler(entityTableName string, operationContainer oc.OperationContainerClient, dbClient *sql.DB) shuttle.HandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) {
		logger := ctxlogger.GetLogger(ctx)

		logger.Info("DeadLetterQueueHandler: Received a message!")
		var body operation.OperationRequest
		err := json.Unmarshal(message.Body, &body)
		if err != nil {
			logger.Error("DeadLetterQueueHandler: Error unmarshalling message: " + err.Error())

			logger.Info("Trying to settle message.")
			err = settler.CompleteMessage(ctx, message, nil)
			if err != nil {
				logger.Error("DeadLetterQueueHandler: Unable to settle unmarshalled message: " + err.Error())
				return
			}
			return
		}

		// Use dbClient to update entity operation state to failed
		query := fmt.Sprintf(`UPDATE %s SET operation_status = @p1 WHERE entity_id = @p2 AND entity_type = @p3 AND last_operation_id = @p4;`, entityTableName)
		failedOperationStatus := oc.Status_FAILED
		_, err = database.ExecDb(ctx, dbClient, query, failedOperationStatus.String(), body.EntityId, body.EntityType, body.OperationId)
		if err != nil {
			logger.Error("DeadLetterQueueHandler: Error in entity table query with operation id: " + body.OperationId + ": " + err.Error())
			return
		}

		// Use operationContainer to update to cancelled.
		updateOperationStatusRequest := &oc.UpdateOperationStatusRequest{
			OperationId: body.OperationId,
			Status:      failedOperationStatus,
		}

		_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
		if err != nil {
			logger.Error("DeadLetterQueueHandler: Error setting operation cancelled in operations table with id: " + body.OperationId + ": " + err.Error())
			return
		}

		// Settle message, which will set is as Complete, and deletes it from the Service Bus Dead Letter Queue.
		logger.Info("Settling message.")
		err = settler.CompleteMessage(ctx, message, nil)
		if err != nil {
			logger.Error("DeadLetterQueueHandler: Unable to settle message with id " + body.OperationId + ": " + err.Error())
			return
		}
		logger.Info("DeadLetterQueueHandler: Successfully set the operation with id " + body.OperationId + " to Failed.")
	}
}
