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

func (h *RunOnlyHooks) BeforeRun(ctx context.Context, op APIOperation) {
	fmt.Println("This is the before internal hook!")
}

func (h *RunOnlyHooks) AfterRun(ctx context.Context, op APIOperation, result Result) {
	fmt.Println("This is the after internal hook!")
}

// Sample operation
var _ APIOperation = &LongRunningOperation{}

type LongRunningOperation struct {
	opReq OperationRequest
	num   int
}

func (l *LongRunningOperation) Run(context.Context) *Result {
	fmt.Println(l.num)
	return &Result{}
}

func (l *LongRunningOperation) GuardConcurrency(context.Context, Entity) (*CategorizedError, error) {
	return nil, nil
}

func (l *LongRunningOperation) Init(ctx context.Context, opReq OperationRequest) (APIOperation, error) {
	l.opReq = opReq
	l.num = 1
	return nil, nil
}

func (l *LongRunningOperation) GetOperationRequest(context.Context) *OperationRequest {
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

	operation, err := matcher.CreateInstance(body.OperationName)
	if err != nil {
		t.Fatalf("Error creating instance of operation: " + err.Error())
	}

	runOnlyHooks := &RunOnlyHooks{}
	hOperation := &HookedApiOperation{
		Operation:      operation,
		OperationHooks: []BaseOperationHooksInterface{runOnlyHooks},
	}

	_, err = hOperation.Init(ctx, body)
	if err != nil {
		t.Fatalf("Error initializing operation: " + err.Error())
	}

	_, err = hOperation.GuardConcurrency(ctx, nil)
	_ = hOperation.Run(ctx)

	fmt.Println("Finished!")
}
