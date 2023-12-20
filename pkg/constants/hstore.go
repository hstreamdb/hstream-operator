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
