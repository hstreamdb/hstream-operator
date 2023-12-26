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

package connectorgen

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

func DefaultSinkElasticsearchContainer(connector *v1beta1.Connector, name, configMapName string) *corev1.Container {
	if configMapName != "" {
		return &corev1.Container{
			Name:  name,
			Image: addImageRegistry(v1beta1.ConnectorImageMap[connector.Spec.Type], connector.Spec.ImageRegistry),
			Args: []string{
				"run",
				"--config /data/config/config.json",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      configMapName,
					MountPath: "/data/config",
				},
				{
					Name:      "data",
					MountPath: "/data",
				},
			},
		}
	}

	return nil
}

func DefaultSinkElasticsearchLogContainer(connector *v1beta1.Connector) corev1.Container {
	return corev1.Container{
		Name:  "log",
		Image: addImageRegistry("busybox:1.36", connector.Spec.ImageRegistry),
		Args: []string{
			"/bin/sh",
			"-c",
			"sleep 5 && tail -F /data/app.log", // OPTIMIZE: wait for connector to start.
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/data",
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("300m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
		},
	}
}
