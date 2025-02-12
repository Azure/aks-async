package operationsbus

import (
	"errors"
	"log/slog"

	sb "github.com/Azure/aks-async/servicebus"

	oc "github.com/Azure/OperationContainer/api/v1"
	"github.com/Azure/go-shuttle/v2"
)

// The processor will be used to process all the operations using the default values or with handlers set by the user.
func CreateProcessor(
	serviceBusReceiver sb.ReceiverInterface,
	matcher *Matcher,
	operationContainer oc.OperationContainerClient,
	entityController EntityController,
	logger *slog.Logger,
	customHandler shuttle.HandlerFunc,
	processorOptions *shuttle.ProcessorOptions,
	hooks []BaseOperationHooksInterface,
) (*shuttle.Processor, error) {

	if serviceBusReceiver == nil {
		return nil, errors.New("No serviceBusReceiver received.")
	}

	if matcher == nil {
		return nil, errors.New("No matcher received.")
	}

	// Define the default handler chain
	// Use the default handler if a custom handler is not provided
	if customHandler == nil {
		customHandler = DefaultHandlers(serviceBusReceiver, matcher, operationContainer, entityController, logger, hooks)
	}

	// Set default processor options.
	if processorOptions == nil {
		processorOptions = &shuttle.ProcessorOptions{
			MaxConcurrency:  1,
			StartMaxAttempt: 5,
		}
	}

	azReceiver, err := serviceBusReceiver.GetAzureReceiver()
	if err != nil {
		return nil, err
	}

	// Create the processor using the (potentially custom) handler
	p := shuttle.NewProcessor(
		azReceiver,
		customHandler,
		processorOptions,
	)

	return p, nil
}
