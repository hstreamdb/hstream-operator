/*
Copyright 2023 HStream Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package constants

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	HServerPortName         = "port"
	HServerInternalPortName = "internal-port"
	HServerGossipPortName   = "gossip-port"
	HServerMetricsPortName  = "metrics-port"
)

var DefaultHServerPort = corev1.ContainerPort{
	Name:          HServerPortName,
	ContainerPort: 6570,
}

var DefaultHServerInternalPort = corev1.ContainerPort{
	Name:          HServerInternalPortName,
	ContainerPort: 6571,
}

var DefaultHServerPorts = []corev1.ContainerPort{
	DefaultHServerPort,
	DefaultHServerInternalPort,
}

var DefaultKafkaHServerPort = corev1.ContainerPort{
	Name:          HServerPortName,
	ContainerPort: 9092,
}

var DefaultKafkaHServerGossipPort = corev1.ContainerPort{
	Name:          HServerGossipPortName,
	ContainerPort: 6571,
}

var DefaultKafkaHServerMetricsPort = corev1.ContainerPort{
	Name:          HServerMetricsPortName,
	ContainerPort: 9700,
}

var DefaultKafkaHServerPorts = []corev1.ContainerPort{
	DefaultKafkaHServerPort,
	DefaultKafkaHServerGossipPort,
	DefaultKafkaHServerMetricsPort,
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
