package controller

import (
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("controller/connector/unit", func() {
	It("should add image registry", func() {
		registry := "hstream.io"
		image := "busybox"

		Expect(addImageRegistry(image, &registry)).To(Equal("hstream.io/busybox"))
		Expect(addImageRegistry(image, nil)).To(Equal("busybox"))
	})

	It("should get prom annotations", func() {
		connector := mock.CreateDefaultConnector("default")

		connector.Annotations = map[string]string{
			"prometheus.io/scrape": "true",
		}
		annotations := getPromAnnotations(connector)

		Expect(annotations).To(Equal(map[string]string{
			"prometheus.io/scrape": "true",
		}))
	})
})
