package constants

import (
	corev1 "k8s.io/api/core/v1"
)

var DefaultHStoreEnv = []corev1.EnvVar{
	{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	},
	{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	},
}

var DefaultHStorePorts = []corev1.ContainerPort{
	{
		Name:          "port",
		ContainerPort: 4440,
		Protocol:      corev1.ProtocolTCP,
	},
	{
		Name:          "gossip-port",
		ContainerPort: 4441,
		Protocol:      corev1.ProtocolTCP,
	},
	{
		Name:          "admin-port",
		ContainerPort: 6440,
		Protocol:      corev1.ProtocolTCP,
	},
}
