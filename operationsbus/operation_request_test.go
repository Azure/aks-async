package operationsbus

import (
	"encoding/json"
	"testing"
)

func TestOperationRequestMarshall(t *testing.T) {
	operation := &OperationRequest{
		OperationName: "LongRunningOperation",
		APIVersion:    "",
		OperationId:   "0",
		Body:          nil,
		HttpMethod:    "",
		RetryCount:    0,
	}
	marshalledOperation, err := json.Marshal(operation)

	if err != nil {
		t.Fatalf("Could not marshall the operation request: " + err.Error())
	}

	var body OperationRequest
	err = json.Unmarshal(marshalledOperation, &body)
	if err != nil {
		t.Fatalf("Could not unmarshall operation request:" + err.Error())
	}
}
