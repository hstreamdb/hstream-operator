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
	"github.com/hstreamdb/hstream-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func DefaultExternalSourceContainer(connector *v1beta1.Connector, name, stream string) *corev1.Container {
	patch, err := connector.GetPatchByStream(stream)
	if err != nil {
		return nil
	}

	args := []string{"-u", connector.Spec.HServerEndpoint}
	for k, v := range patch {
		args = append(args, k, v.(string))
	}

	return &corev1.Container{
		Name:  name,
		Image: addImageRegistry(v1beta1.ConnectorImageMap[connector.Spec.Type], connector.Spec.ImageRegistry),
		Args:  args,
	}
}
