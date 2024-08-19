package operationsbus

import (
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Sample struct {
	Message string
	Num     int
}

func TestOperationRequestMarshall(t *testing.T) {
	// matcher := NewMatcher()

	expirationTime := timestamppb.New(time.Now().Add(1 * time.Hour))
	extension := &Sample{
		Message: "Hello",
		Num:     1,
	}
	operation := NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, expirationTime, nil, "", extension)
	marshalledOperation, err := json.Marshal(operation)

	if err != nil {
		t.Fatalf("Could not marshall the operation request: " + err.Error())
	}

	var body OperationRequest
	err = json.Unmarshal(marshalledOperation, &body)
	if err != nil {
		t.Fatalf("Could not unmarshall operation request:" + err.Error())
	}

	// Test getting the extension
	s := &Sample{}
	err = body.SetExtension(s)
	if err != nil {
		t.Fatalf("SetExtension errored: " + err.Error())
	}

	// Check if the type and value are correctly set
	if ext, ok := body.Extension.(*Sample); ok {
		if ext.Message != "Hello" || ext.Num != 1 {
			t.Fatalf("Extension data does not match. Got %+v", ext)
		}
	} else {
		t.Fatalf("Extension is not of type *Sample")
	}
}
