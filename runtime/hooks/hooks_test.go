package hooks

import (
	"context"
	"testing"

	"github.com/Azure/aks-async/runtime/errors"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hooks Suite")
}

// Sample hook
type RunOnlyHooks struct {
	HookedApiOperation
}

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op operation.ApiOperation) *errors.AsyncError {
	if sampleOp, ok := (op).(*sampleOperation.SampleOperation); ok {
		sampleOp.Num += 1
	}
	return nil
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op operation.ApiOperation, err *errors.AsyncError) *errors.AsyncError {
	if sampleOp, ok := (op).(*sampleOperation.SampleOperation); ok {
		sampleOp.Num += 1
	}
	return nil
}

var _ = Describe("Hooks", func() {
	var (
		ctx               context.Context
		opRequest         *operation.OperationRequest
		operationInstance *sampleOperation.SampleOperation
		runOnlyHooks      *RunOnlyHooks
		hooksSlice        []BaseOperationHooksInterface
		hOperation        *HookedApiOperation
	)

	BeforeEach(func() {
		ctx = context.Background()
		opRequest = &operation.OperationRequest{
			OperationName:       "SampleOperation",
			ApiVersion:          "v0.0.1",
			OperationId:         "0",
			EntityId:            "1",
			EntityType:          "Cluster",
			RetryCount:          0,
			ExpirationTimestamp: nil,
			Body:                nil,
			HttpMethod:          "",
			Extension:           nil,
		}
		operationInstance = &sampleOperation.SampleOperation{}
		runOnlyHooks = &RunOnlyHooks{}
		hooksSlice = []BaseOperationHooksInterface{runOnlyHooks}
		hOperation = &HookedApiOperation{
			OperationInstance: operationInstance,
			OperationHooks:    hooksSlice,
		}
		_, err := hOperation.InitOperation(ctx, *opRequest)
		Expect(err).NotTo(HaveOccurred())
	})

	//TODO(mheberling): Add handling of errors in hooks.
	It("should run hooks successfully", func() {
		_ = hOperation.GuardConcurrency(ctx, nil)
		_ = hOperation.Run(ctx)
		if sampleOp, ok := (hOperation.OperationInstance).(*sampleOperation.SampleOperation); ok {
			Expect(sampleOp.Num).To(Equal(3))
		} else {
			Fail("Something went wrong casting the operation to LongRunningOperation type.")
		}
	})
})
