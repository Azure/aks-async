package hooks

import (
	"context"
	// "encoding/json"

	"testing"

	"github.com/Azure/aks-async/runtime/entity"
	// "github.com/Azure/aks-async/runtime/matcher"
	"github.com/Azure/aks-async/runtime/operation"
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

func TestHooks(t *testing.T) {
	ctx := context.Background()

	// matcher := matcher.NewMatcher()
	// lro := &LongRunningOperation{}
	// matcher.Register("LongRunningOperation", lro)

	opRequest := operation.NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, nil, nil, "", nil)
	operationInstance := &LongRunningOperation{}
	// marshalledOperation, err := json.Marshal(opRequest)
	// if err != nil {
	// 	t.Fatalf("Error marshalling operation.")
	// }
	//
	// var body operation.OperationRequest
	// err = json.Unmarshal(marshalledOperation, &body)
	// if err != nil {
	// 	t.Fatalf("Error unmarshalling operation: " + err.Error())
	// }
	//
	// // Testing with a regular instance
	// operationInstance, err := matcher.CreateOperationInstance(body.OperationName)
	// if err != nil {
	// 	t.Fatalf("Error creating instance of operation: " + err.Error())
	// }

	runOnlyHooks := &RunOnlyHooks{}
	hooksSlice := []BaseOperationHooksInterface{runOnlyHooks}
	hOperation := &HookedApiOperation{
		OperationInstance: operationInstance,
		OperationHooks:    hooksSlice,
	}

	_, err := hOperation.InitOperation(ctx, *opRequest)
	if err != nil {
		t.Fatalf("Error initializing operation: " + err.Error())
	}

	_ = hOperation.GuardConcurrency(ctx, nil)
	_ = hOperation.Run(ctx)
	if longOp, ok := (hOperation.OperationInstance).(*LongRunningOperation); ok {
		if longOp.num == 3 {
			t.Log("Hooks did ran successfully.")
		} else {
			t.Fatalf("Hooks did not run successfully.")
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunningOperation type.")
	}

	// err = json.Unmarshal(marshalledOperation, *opRequest)
	// if err != nil {
	// 	t.Fatalf("Error unmarshalling operation: " + err.Error())
	// }
	//
	// // Testing with a Hooked Instace
	// hOperation, err = matcher.CreateHookedInstace(body.OperationName, hooksSlice)
	// if err != nil {
	// 	t.Fatalf("Error creating instance of operation: " + err.Error())
	// }
	//
	// _, err = hOperation.InitOperation(ctx, body)
	// if err != nil {
	// 	t.Fatalf("Error initializing operation: " + err.Error())
	// }
	//
	// _ = hOperation.GuardConcurrency(ctx, nil)
	// _ = hOperation.Run(ctx)
	// if longOp, ok := (hOperation.OperationInstance).(*LongRunningOperation); ok {
	// 	if longOp.num == 3 {
	// 		t.Log("Hooks did ran successfully.")
	// 	} else {
	// 		t.Fatalf("Hooks did not run successfully.")
	// 	}
	// } else {
	// 	t.Fatalf("Something went wrong casting the operation to LongRunningOperation type.")
	// }
}
