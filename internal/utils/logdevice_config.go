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
	"bytes"
	"text/template"

	"github.com/Jeffail/gabs/v2"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	LogDeviceConfigKey               = "config.json"
	defaultRecommendedLogReplication = 3
)

const defaultLogDeviceConfigTemplate = `{
	"server_settings": {
		"enable-node-self-registration":                       "true",
		"enable-nodes-configuration-manager":                  "true",
		"enable-cluster-maintenance-state-machine":            "true",
		"use-nodes-configuration-manager-nodes-configuration": "true"
	},
	"client_settings": {
		"enable-nodes-configuration-manager": "true",
		"use-nodes-configuration-manager-nodes-configuration": "true",
		"admin-client-capabilities": "true"
	},
	"cluster": "hstore",
	"internal_logs": {
		"config_log_deltas": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		},
		"config_log_snapshots": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		},
		"event_log_deltas": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		},
		"event_log_snapshots": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		},
		"maintenance_log_deltas": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		},
		"maintenance_log_snapshots": {
			"replicate_across": {
				"node": {{ .NodeNum }}
			}
		}
	},
	"rqlite": {
		"rqlite_uri": "ip://{{ .HMetaAddr }}"
	},
	"version": 1
}`

func GetRecommendedLogReplicaAcross(hdb *hapi.HStreamDB) int {
	replicas := hdb.Spec.HStore.Replicas

	if replicas <= defaultRecommendedLogReplication {
		return int(replicas)
	}

	return defaultRecommendedLogReplication
}

func GetLogDeviceConfig(nodeNum int32, hmetaAddr string, existingConfig []byte) (string, error) {
	tmpl, err := template.New("defaultLogDeviceConfig").Parse(defaultLogDeviceConfigTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, map[string]any{
		"NodeNum":   nodeNum,
		"HMetaAddr": hmetaAddr,
	})

	jsonParsed, err := gabs.ParseJSON(buf.Bytes())
	if err != nil {
		return "", err
	}

	if len(existingConfig) > 0 {
		existingParsed, err := gabs.ParseJSON(existingConfig)
		if err != nil {
			return "", err
		}

		jsonParsed.Merge(existingParsed)
	}

	return jsonParsed.String(), nil
}

func GetLogDeviceConfigMapNamespacedName(hdb *hapi.HStreamDB) types.NamespacedName {
	return types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hdb.Name + "-logdevice-config",
	}
}

func GetLogDeviceConfigVolume(hdb *hapi.HStreamDB) corev1.Volume {
	name := GetLogDeviceConfigMapNamespacedName(hdb).Name

	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
				Items: []corev1.KeyToPath{
					{
						Key:  LogDeviceConfigKey,
						Path: LogDeviceConfigKey,
					},
				},
			},
		},
	}
}

func GetLogDeviceConfigVolumeMount(hdb *hapi.HStreamDB) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      GetLogDeviceConfigMapNamespacedName(hdb).Name,
		MountPath: "/etc/logdevice",
		ReadOnly:  true,
	}
}
