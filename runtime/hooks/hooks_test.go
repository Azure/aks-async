package hooks

import (
	"context"
	"testing"

	// "github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
	sampleOperation "github.com/Azure/aks-async/runtime/testutils/operation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Sample hook
type RunOnlyHooks struct {
	HookedApiOperation
}

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op operation.ApiOperation) error {
	if sampleOp, ok := (op).(*sampleOperation.SampleOperation); ok {
		sampleOp.Num += 1
	}
	return nil
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op operation.ApiOperation, err error) error {
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
		opRequest = operation.NewOperationRequest("SampleOperation", "v0.0.1", "0", "1", "Cluster", 0, nil, nil, "", nil)
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

func TestHooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hooks Suite")
}
