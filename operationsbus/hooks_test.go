package operationsbus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// Sample hook
type RunOnlyHooks struct {
	HookedApiOperation
}

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op *ApiOperation) error {
	fmt.Println("This is the before internal run hook!")
	if longOp, ok := (*op).(*LongRunningOperation); ok {
		longOp.num += 1
	}
	return nil
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op *ApiOperation, err error) error {
	fmt.Println("This is the after internal run hook!")
	if longOp, ok := (*op).(*LongRunningOperation); ok {
		longOp.num += 1
	}
	return nil
}

// Sample operation
var _ ApiOperation = &LongRunningOperation{}

type LongRunningOperation struct {
	opReq OperationRequest
	num   int
}

func (l *LongRunningOperation) Run(context.Context) error {
	return nil
}

func (l *LongRunningOperation) GuardConcurrency(context.Context) *CategorizedError {
	return nil
}

func (l *LongRunningOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, error) {
	l.opReq = opReq
	l.num = 1
	return nil, nil
}

func (l *LongRunningOperation) GetOperationRequest() *OperationRequest {
	return &l.opReq
}

func TestHooks(t *testing.T) {
	fmt.Println("Hello world!")
	ctx := context.Background()

	matcher := NewMatcher()
	lro := &LongRunningOperation{}
	matcher.Register("LongRunningOperation", lro)

	opRequest := NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, nil, nil, "", nil)

	marshalledOperation, err := json.Marshal(opRequest)
	if err != nil {
		t.Fatalf("Error marshalling operation.")
	}

	var body OperationRequest
	err = json.Unmarshal(marshalledOperation, &body)
	if err != nil {
		t.Fatalf("Error unmarshalling operation: " + err.Error())
	}

	// Testing with a regular instance
	operation, err := matcher.CreateInstance(body.OperationName)
	if err != nil {
		t.Fatalf("Error creating instance of operation: " + err.Error())
	}

	runOnlyHooks := &RunOnlyHooks{}
	hooksSlice := []BaseOperationHooksInterface{runOnlyHooks}
	hOperation := &HookedApiOperation{
		Operation:      &operation,
		OperationHooks: hooksSlice,
	}

	_, err = hOperation.InitOperation(ctx, body)
	if err != nil {
		t.Fatalf("Error initializing operation: " + err.Error())
	}

	_ = hOperation.GuardConcurrency(ctx)
	_ = hOperation.Run(ctx)
	if longOp, ok := (*hOperation.Operation).(*LongRunningOperation); ok {
		if longOp.num == 3 {
			t.Log("Hooks did ran successfully.")
		} else {
			t.Fatalf("Hooks did not run successfully.")
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunningOperation type.")
	}

	err = json.Unmarshal(marshalledOperation, &body)
	if err != nil {
		t.Fatalf("Error unmarshalling operation: " + err.Error())
	}

	// Testing with a Hooked Instace
	hOperation, err = matcher.CreateHookedInstace(body.OperationName, hooksSlice)
	if err != nil {
		t.Fatalf("Error creating instance of operation: " + err.Error())
	}

	_, err = hOperation.InitOperation(ctx, body)
	if err != nil {
		t.Fatalf("Error initializing operation: " + err.Error())
	}

	_ = hOperation.GuardConcurrency(ctx)
	_ = hOperation.Run(ctx)
	if longOp, ok := (*hOperation.Operation).(*LongRunningOperation); ok {
		if longOp.num == 3 {
			t.Log("Hooks did ran successfully.")
		} else {
			t.Fatalf("Hooks did not run successfully.")
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunningOperation type.")
	}
}
