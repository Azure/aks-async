package qos

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQoSHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QoSHandler Suite")
}
