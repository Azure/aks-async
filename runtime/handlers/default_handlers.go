package handlers

import (
	"log/slog"
	"time"

	oc "github.com/Azure/OperationContainer/api/v1"
	ec "github.com/Azure/aks-async/runtime/entity_controller"
	"github.com/Azure/aks-async/runtime/handlers/errors"
	"github.com/Azure/aks-async/runtime/handlers/log"
	"github.com/Azure/aks-async/runtime/handlers/operation"
	och "github.com/Azure/aks-async/runtime/handlers/operation_container"
	"github.com/Azure/aks-async/runtime/handlers/qos"
	"github.com/Azure/aks-async/runtime/hooks"
	"github.com/Azure/aks-async/runtime/matcher"
	sb "github.com/Azure/aks-async/servicebus"
	"github.com/Azure/go-shuttle/v2"
)

func DefaultHandlers(
	serviceBusReceiver sb.ReceiverInterface,
	matcher *matcher.Matcher,
	operationContainer oc.OperationContainerClient,
	entityController ec.EntityController,
	logger *slog.Logger,
	hooks []hooks.BaseOperationHooksInterface,
	marshaller shuttle.Marshaller,
) shuttle.HandlerFunc {

	// Lock renewal settings
	lockRenewalInterval := 10 * time.Second
	lockRenewalOptions := &shuttle.LockRenewalOptions{Interval: &lockRenewalInterval}

	if marshaller == nil {
		marshaller = &shuttle.DefaultProtoMarshaller{}
	}

	var errorHandler errors.ErrorHandlerFunc
	if operationContainer != nil {
		errorHandler = och.NewOperationContainerHandler(
			errors.NewErrorReturnHandler(
				operation.NewOperationHandler(matcher, hooks, entityController, marshaller),
				nil,
				marshaller,
			),
			operationContainer,
			marshaller,
		)
	} else {
		errorHandler = errors.NewErrorReturnHandler(
			operation.NewOperationHandler(matcher, hooks, entityController, marshaller),
			nil,
			marshaller,
		)
	}

	// Combine handlers into a single default handler
	return shuttle.NewPanicHandler(
		nil,
		shuttle.NewRenewLockHandler(
			lockRenewalOptions,
			log.NewLogHandler(
				logger,
				qos.NewQosErrorHandler(
					errorHandler,
				),
				marshaller,
			),
		),
	)
}
