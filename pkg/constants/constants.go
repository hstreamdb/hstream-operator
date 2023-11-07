package constants

import (
	corev1 "k8s.io/api/core/v1"
)

var DefaultHServerPort = corev1.ContainerPort{
	Name:          "port",
	ContainerPort: 6570,
	Protocol:      corev1.ProtocolTCP,
}

var DefaultHServerInternalPort = corev1.ContainerPort{
	Name:          "internal-port",
	ContainerPort: 6571,
	Protocol:      corev1.ProtocolTCP,
}

var DefaultHMetaPort = corev1.ContainerPort{
	Name:          "rqlite",
	ContainerPort: 4001,
	Protocol:      corev1.ProtocolTCP,
}
