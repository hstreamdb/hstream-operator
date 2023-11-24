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

var DefaultHServerEnv = []corev1.EnvVar{
	{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	},
}
