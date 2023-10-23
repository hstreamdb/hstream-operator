/*
Copyright 2023.

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

package mock

import (
	"github.com/hstreamdb/hstream-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDefaultConnectorTemplate() v1beta1.ConnectorTemplate {
	return v1beta1.ConnectorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "connector-template-sample",
			Namespace: "default",
		},
		Spec: v1beta1.ConnectorTemplateSpec{
			Type: "sink-elasticsearch", // We only support this type for now.
			Config: `
			{
				"auth": "basic",
				"username": "elastic",
				"password": "elastic",
				"enableLogReport": true,
				"buffer.enableBackgroundFlush": false,
				"buffer.batch.maxAge": 0,
				"buffer.batch.maxBytesSize": 0,
				"scheme": "http",
				"hosts": "localhost:9200",
				"task.error.maxRetries": 3,
				"task.error.skipStrategy": "SkipAll",
				"task.reader.fromOffset": "EARLIEST"
			}`,
		},
	}
}
