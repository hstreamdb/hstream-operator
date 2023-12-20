package constants

import (
	corev1 "k8s.io/api/core/v1"
)

var DefaultHMetaPort = corev1.ContainerPort{
	Name:          "rqlite",
	ContainerPort: 4001,
	Protocol:      corev1.ProtocolTCP,
}
