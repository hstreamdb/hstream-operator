package connectorgen

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("connectorgen/utils", func() {
	It("should add image registry", func() {
		registry := "hstream.io"
		image := "busybox"
		Expect(addImageRegistry(image, &registry)).To(Equal("hstream.io/busybox"))
		Expect(addImageRegistry(image, nil)).To(Equal("busybox"))
	})
})
