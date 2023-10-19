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

package v1beta1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenConnectorConfigMapName(connectorName string, isConnectorTemplate bool) (suffix string) {
	suffix += connectorName + "-hstream-io"

	if isConnectorTemplate {
		suffix += "-connector-template-config"
	} else {
		suffix += "-connector-config"
	}

	return
}

// DeleteAssociatedConfigMap deletes the ConfigMap which generated by Connectors or ConnectorTemplates.
func DeleteAssociatedConfigMap(ctx context.Context, c client.Client, namespacedName types.NamespacedName) error {
	var configMap corev1.ConfigMap
	if err := c.Get(ctx, namespacedName, &configMap); err != nil {
		return err
	}

	return c.Delete(ctx, &configMap)
}

func GenConnectorDeploymentName(connectorName, stream string) string {
	return connectorName + "-" + stream + "-hstream-io-connector-depolyment"
}