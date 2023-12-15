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

package utils

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const NShardsConfigKey = "NSHARDS"

func GetMinNShards(hdb *hapi.HStreamDB) int {
	if hdb.Spec.Config.NShards == 0 {
		return 1
	}

	return int(hdb.Spec.Config.NShards)
}

func GetNShardsConfigMapNamespacedName(hdb *hapi.HStreamDB) types.NamespacedName {
	return types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hdb.Name + "-nshards",
	}
}

func GetNShardsConfigVolume(hdb *hapi.HStreamDB) corev1.Volume {
	name := GetNShardsConfigMapNamespacedName(hdb).Name

	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
				Items: []corev1.KeyToPath{
					{
						Key:  NShardsConfigKey,
						Path: NShardsConfigKey,
					},
				},
			},
		},
	}
}

func GetNShardsConfigVolumeMount(hdb *hapi.HStreamDB) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      GetNShardsConfigMapNamespacedName(hdb).Name,
		MountPath: "/data/logdevice",
		ReadOnly:  true,
	}
}
