package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("internal/utils/logdevice_config", func() {
	It("should generate correct default logdevice config", func() {
		config, err := GetLogDeviceConfig(3, "internal-rqlite.hstreamdb:80", []byte("{}"))

		Expect(err).To(BeNil())
		Expect(config).ToNot(Equal(""))
	})
})
