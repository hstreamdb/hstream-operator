package connectorgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectorgen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connectorgen Suite")
}
