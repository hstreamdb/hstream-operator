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

func CreateDefaultConnector(ns string) v1beta1.Connector {
	return v1beta1.Connector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-connector",
			Namespace: ns,
		},
		Spec: v1beta1.ConnectorSpec{
			Type:         "sink-elasticsearch", // We only support this type for now.
			TemplateName: "test-connector-template",
			Streams: []string{
				"stream01",
			},
		},
	}
}
