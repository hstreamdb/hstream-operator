package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("internal/utils/logdevice_config", func() {
	It("should generate correct default logdevice config", func() {
		config, err := GetLogDeviceConfig(3, "hmeta.default:4001", []byte("{}"))

		Expect(err).To(BeNil())
		Expect(config).ToNot(Equal(""))
	})
})
