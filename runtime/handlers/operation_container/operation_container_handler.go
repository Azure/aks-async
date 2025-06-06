package operation

import (
	"context"

	oc "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/aks-async/runtime/errors"
	errorHandlers "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// Handler for when the user uses the OperationContainer.
func NewOperationContainerHandler(errHandler errorHandlers.ErrorHandlerFunc, operationContainer oc.OperationContainerClient, marshaller shuttle.Marshaller) errorHandlers.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *errors.AsyncError {
		logger := ctxlogger.GetLogger(ctx)

		var body operation.OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			logger.Error("OperationContainerHandler: Error unmarshalling message: " + err.Error())
			return nil
		}

		updateOperationStatusRequest := &oc.UpdateOperationStatusRequest{
			OperationId: body.OperationId,
			Status:      oc.Status_IN_PROGRESS,
		}
		_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
		if err != nil {
			errorMessage := "OperationContainerHandler: Error setting operation in progress: " + err.Error()
			logger.Error(errorMessage)
			return &errors.AsyncError{
				OriginalError: err,
				Message:       errorMessage,
				ErrorCode:     500,
			}
		}
		asyncErr := errHandler.Handle(ctx, settler, message)

		if asyncErr != nil {
			logger.Info("OperationContainerHandler: Handling error: " + asyncErr.Error())
			switch asyncErr.OriginalError.(type) {
			case *errors.NonRetryError:
				// Fail the operation
				logger.Info("OperationContainerHandler: Setting operation as Failed.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_FAILED,
				}
				_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if err != nil {
					errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Failed" + err.Error()
					logger.Error(errorMessage)
					return &errors.AsyncError{
						OriginalError: asyncErr,
						Message:       errorMessage,
						ErrorCode:     500,
					}
				}
			case *errors.RetryError:
				// Set the operation as Pending
				logger.Info("OperationContainerHandler: Setting operation as Pending.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_PENDING,
				}
				_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if err != nil {
					errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Pending:" + err.Error()
					logger.Error(errorMessage)
					return &errors.AsyncError{
						OriginalError: asyncErr,
						Message:       errorMessage,
						ErrorCode:     500,
					}
				}
			default:
				errorMessage := "OperationContainerHandler: Error type not recognized. Operation status not changed."
				logger.Info(errorMessage)
				return &errors.AsyncError{
					OriginalError: asyncErr,
					Message:       errorMessage,
					ErrorCode:     500,
				}
			}
			return asyncErr
		} else {
			logger.Info("OperationContainerHandler: Setting Operation as Succeeded.")
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: body.OperationId,
				Status:      oc.Status_SUCCEEDED,
			}
			_, updateErr := operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			if updateErr != nil {
				errorMessage := "OperationContainerHandler: Something went wrong setting the operation as Completed:" + updateErr.Error()
				logger.Info(errorMessage)
				return &errors.AsyncError{
					OriginalError: updateErr,
					Message:       errorMessage,
					ErrorCode:     500,
				}
			}
		}

		return nil
	}
}
