package operation

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Sample struct {
	Message string
	Num     int
}

var _ = Describe("OperationRequest", func() {
	var (
		expirationTime      *timestamppb.Timestamp
		extension           *Sample
		operation           *OperationRequest
		marshalledOperation []byte
		err                 error
		body                OperationRequest
	)

	BeforeEach(func() {
		expirationTime = timestamppb.New(time.Now().Add(1 * time.Hour))
		extension = &Sample{
			Message: "Hello",
			Num:     1,
		}
		operation = NewOperationRequest("LongRunningOperation", "v0.0.1", "1", "1", "Cluster", 0, expirationTime, nil, "", extension)
		marshalledOperation, err = json.Marshal(operation)
	})

	It("should marshal the operation request without error", func() {
		Expect(err).NotTo(HaveOccurred(), "Could not marshall the operation request")
	})

	It("should unmarshal the operation request without error", func() {
		err = json.Unmarshal(marshalledOperation, &body)
		Expect(err).NotTo(HaveOccurred(), "Could not unmarshall operation request")
	})

	It("should set and get the extension correctly", func() {
		s := &Sample{}
		err = body.SetExtension(s)
		Expect(err).NotTo(HaveOccurred(), "SetExtension errored")

		ext, ok := body.Extension.(*Sample)
		Expect(ok).To(BeTrue(), "Extension is not of type *Sample")
		Expect(ext.Message).To(Equal("Hello"), "Extension data does not match")
		Expect(ext.Num).To(Equal(1), "Extension data does not match")
	})
})

func TestOperationRequestMarshall(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperationRequest Suite")
}
