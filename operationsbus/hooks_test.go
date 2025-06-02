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

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op ApiOperation) *AsyncError {
	fmt.Println("This is the before internal run hook!")
	if longOp, ok := (op).(*LongRunningOperation); ok {
		longOp.num += 1
	}
	return nil
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op ApiOperation, err *AsyncError) *AsyncError {
	fmt.Println("This is the after internal run hook!")
	if longOp, ok := (op).(*LongRunningOperation); ok {
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

func (l *LongRunningOperation) InitOperation(ctx context.Context, opReq OperationRequest) (ApiOperation, *AsyncError) {
	l.opReq = opReq
	l.num = 1
	return nil, nil
}

func (l *LongRunningOperation) GuardConcurrency(ctx context.Context, entity Entity) *AsyncError {
	return nil
}

func (l *LongRunningOperation) Run(ctx context.Context) *AsyncError {
	return nil
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

	opRequest := &OperationRequest{
		OperationName:       "LongRunningOperation",
		ApiVersion:          "v0.0.1",
		RetryCount:          0,
		OperationId:         "1",
		EntityId:            "1",
		EntityType:          "Cluster",
		ExpirationTimestamp: nil,
		Body:                nil,
		HttpMethod:          "",
		Extension:           nil,
	}

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
	operation, err := matcher.CreateOperationInstance(body.OperationName)
	if err != nil {
		t.Fatalf("Error creating instance of operation: " + err.Error())
	}

	runOnlyHooks := &RunOnlyHooks{}
	hooksSlice := []BaseOperationHooksInterface{runOnlyHooks}
	hOperation := &HookedApiOperation{
		Operation:      operation,
		OperationHooks: hooksSlice,
	}

	_, asyncErr := hOperation.InitOperation(ctx, body)
	if asyncErr != nil {
		t.Fatalf("Error initializing operation: " + asyncErr.Error())
	}

	_ = hOperation.GuardConcurrency(ctx, nil)
	_ = hOperation.Run(ctx)
	if longOp, ok := (hOperation.Operation).(*LongRunningOperation); ok {
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

	_, asyncErr = hOperation.InitOperation(ctx, body)
	if asyncErr != nil {
		t.Fatalf("Error initializing operation: " + asyncErr.Error())
	}

	_ = hOperation.GuardConcurrency(ctx, nil)
	_ = hOperation.Run(ctx)
	if longOp, ok := (hOperation.Operation).(*LongRunningOperation); ok {
		if longOp.num == 3 {
			t.Log("Hooks did ran successfully.")
		} else {
			t.Fatalf("Hooks did not run successfully.")
		}
	} else {
		t.Fatalf("Something went wrong casting the operation to LongRunningOperation type.")
	}
}
