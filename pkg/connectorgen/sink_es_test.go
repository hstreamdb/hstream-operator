package connectorgen

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

var _ = Describe("connectorgen/sink_es", func() {
	It("should generate default sink elasticsearch container", func() {
		connector := &v1beta1.Connector{
			Spec: v1beta1.ConnectorSpec{
				Type: "sink-elasticsearch",
			},
		}
		container := DefaultSinkElasticsearchContainer(connector, "test", "test")

		Expect(container.Name).To(Equal("test"))
		Expect(container.Image).To(Equal("hstreamdb/sink-elasticsearch:standalone"))
		Expect(container.Args).To(Equal([]string{
			"run",
			"--config /data/config/config.json",
		}))
		Expect(container.VolumeMounts).To(Equal([]corev1.VolumeMount{
			{
				Name:      "test",
				MountPath: "/data/config",
			},
			{
				Name:      "data",
				MountPath: "/data",
			},
		}))
	})
})
