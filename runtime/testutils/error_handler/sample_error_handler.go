package error_handler

import (
	"context"

	asyncErrors "github.com/Azure/aks-async/runtime/errors"
	handlerErrors "github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-shuttle/v2"
)

func SampleErrorHandler(testError error) handlerErrors.ErrorHandlerFunc {
	return func(ctx context.Context, settler shuttle.MessageSettler, message *azservicebus.ReceivedMessage) *asyncErrors.AsyncError {
		return &asyncErrors.AsyncError{OriginalError: testError}
	}
}
