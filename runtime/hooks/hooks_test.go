package hooks

import (
	"context"
	"testing"

	"github.com/Azure/aks-async/runtime/entity"
	"github.com/Azure/aks-async/runtime/operation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Sample hook
type RunOnlyHooks struct {
	HookedApiOperation
}

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op operation.ApiOperation) error {
	if longOp, ok := (op).(*LongRunningOperation); ok {
		longOp.num += 1
	}
	return nil
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op operation.ApiOperation, err error) error {
	if longOp, ok := (op).(*LongRunningOperation); ok {
		longOp.num += 1
	}
	return nil
}

// Sample operation
var _ operation.ApiOperation = &LongRunningOperation{}

type LongRunningOperation struct {
	opReq operation.OperationRequest
	num   int
}

func (l *LongRunningOperation) Run(ctx context.Context) error {
	return nil
}

func (l *LongRunningOperation) GuardConcurrency(ctx context.Context, entityInstance entity.Entity) *entity.CategorizedError {
	return nil
}

func (l *LongRunningOperation) InitOperation(ctx context.Context, opReq operation.OperationRequest) (operation.ApiOperation, error) {
	l.opReq = opReq
	l.num = 1
	return nil, nil
}

func (l *LongRunningOperation) GetOperationRequest() *operation.OperationRequest {
	return &l.opReq
}

var _ = Describe("Hooks", func() {
	var (
		ctx               context.Context
		opRequest         *operation.OperationRequest
		operationInstance *LongRunningOperation
		runOnlyHooks      *RunOnlyHooks
		hooksSlice        []BaseOperationHooksInterface
		hOperation        *HookedApiOperation
		err               error
	)

	BeforeEach(func() {
		ctx = context.Background()
		opRequest = operation.NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, nil, nil, "", nil)
		operationInstance = &LongRunningOperation{}
		runOnlyHooks = &RunOnlyHooks{}
		hooksSlice = []BaseOperationHooksInterface{runOnlyHooks}
		hOperation = &HookedApiOperation{
			OperationInstance: operationInstance,
			OperationHooks:    hooksSlice,
		}
		_, err = hOperation.InitOperation(ctx, *opRequest)
	})

	It("should initialize operation without error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should run hooks successfully", func() {
		_ = hOperation.GuardConcurrency(ctx, nil)
		_ = hOperation.Run(ctx)
		if longOp, ok := (hOperation.OperationInstance).(*LongRunningOperation); ok {
			Expect(longOp.num).To(Equal(3))
		} else {
			Fail("Something went wrong casting the operation to LongRunningOperation type.")
		}
	})
})

func TestHooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hooks Suite")
}
