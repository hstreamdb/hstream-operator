package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("internal/utils/common", func() {
	ports := []corev1.ContainerPort{
		{
			Name:          "port",
			ContainerPort: 6570,
		},
		{
			Name:          "internal-port",
			ContainerPort: 6571,
		},
	}

	It("should merge two container ports", func() {
		mergedPorts := MergeContainerPorts(ports, corev1.ContainerPort{
			Name:          "port",
			ContainerPort: 6572,
		})

		Expect(mergedPorts[0].ContainerPort).To(Equal(int32(6572)))
	})

	It("should append a new port", func() {
		mergedPorts := MergeContainerPorts(ports, corev1.ContainerPort{
			Name:          "new-port",
			ContainerPort: 6700,
		})

		Expect(len(mergedPorts)).To(Equal(3))
		Expect(mergedPorts[2].Name).To(Equal("new-port"))
	})
})
