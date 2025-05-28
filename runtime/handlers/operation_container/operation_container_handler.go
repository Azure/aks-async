package operation

import (
	"context"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	oc "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/operation"
	"github.com/Azure/aks-middleware/grpc/server/ctxlogger"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

// Handler for when the user uses the OperationContainer
func NewOperationContainerHandler(errHandler errors.ErrorHandlerFunc, operationContainer oc.OperationContainerClient, marshaller shuttle.Marshaller) errors.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) error {
		logger := ctxlogger.GetLogger(ctx)

		var body operation.OperationRequest
		err := marshaller.Unmarshal(message.Message(), &body)
		if err != nil {
			logger.Error("OperationContainerHandler: Error unmarshalling message: " + err.Error())
			return nil
		}

		//TODO(mheberling): Update this to not retry, since the service itself should retry if it faces an error not us.
		var updateOperationStatusRequest *oc.UpdateOperationStatusRequest
		// If the operation is picked up immediately from the service bus, while the operationContainer is still putting the
		// operation into the hcp and operations databases, this step might fail if both databases have not been updated.
		// Allowing a couple of retries before fully failing the operation due to this error.
		opInProgress := false
		for i := 0; i < 5; i++ {
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: body.OperationId,
				Status:      oc.Status_IN_PROGRESS,
			}
			_, err = operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			if err != nil {
				logger.Error("OperationContainerHandler: Error setting operation in progress: " + err.Error())
				backoff := exponentialBackoff(i)
				logger.Info("Trying again.")
				logger.Info("Retry %d: backoff for %v\n", strconv.Itoa(i), backoff)
				time.Sleep(backoff)
			} else {
				opInProgress = true
				break
			}
		}

		if !opInProgress {
			logger.Error("Operation was not able to be put in progress.")
			return err
		}

		err = errHandler.Handle(ctx, settler, message)

		if err != nil {
			logger.Info("OperationContainerHandler: Handling error: " + err.Error())
			switch err.(type) {
			case *errors.NonRetryError:
				// Cancel the operation
				logger.Info("OperationContainerHandler: Setting operation as Cancelled.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_CANCELLED,
				}
				_, updateErr := operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if updateErr != nil {
					logger.Error("OperationContainerHandler: Something went wrong setting the operation as Cancelled" + err.Error())
					return updateErr
				}
			case *errors.RetryError:
				// Set the operation as Pending
				logger.Info("OperationContainerHandler: Setting operation as Pending.")
				updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
					OperationId: body.OperationId,
					Status:      oc.Status_PENDING,
				}
				_, updateErr := operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
				if updateErr != nil {
					logger.Error("OperationContainerHandler: Something went wrong setting the operation as Pending:" + err.Error())
					return updateErr
				}
			default:
				logger.Info("OperationContainerHandler: Error type not recognized. Operation status not changed.")
			}
		} else {
			logger.Info("Setting Operation as Successful.")
			updateOperationStatusRequest = &oc.UpdateOperationStatusRequest{
				OperationId: body.OperationId,
				Status:      oc.Status_COMPLETED,
			}
			_, updateErr := operationContainer.UpdateOperationStatus(ctx, updateOperationStatusRequest)
			if updateErr != nil {
				logger.Error("OperationContainerHandler: Something went wrong setting the operation as Completed: " + updateErr.Error())
				return updateErr
			}
		}

		return err
	}
}

// ExponentialBackoff calculates the backoff duration based on the retry count.
func exponentialBackoff(retry int) time.Duration {
	min := 100 * time.Millisecond
	max := 10 * time.Second
	factor := 2.0
	jitter := 0.5

	// Calculate exponential backoff with jitter
	backoff := float64(min) * math.Pow(factor, float64(retry))
	backoff = backoff * (1 + jitter*(rand.Float64()*2-1))

	// Ensure the backoff is within the min and max bounds
	if backoff > float64(max) {
		backoff = float64(max)
	} else if backoff < float64(min) {
		backoff = float64(min)
	}

	return time.Duration(backoff)
}
