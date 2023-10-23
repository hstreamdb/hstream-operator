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

package controller

import (
	"context"

	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

var _ = Describe("controller/connector", func() {
	connectorTpl := mock.CreateDefaultConnectorTemplate()
	connector := mock.CreateDefaultConnector()

	It("should create a connector successfully", func() {
		By("creating a connector template")
		ctx := context.Background()

		Expect(k8sClient.Create(ctx, &connectorTpl)).Should(Succeed())

		By("creating a connector")
		ctx = context.Background()

		Expect(k8sClient.Create(ctx, &connector)).Should(Succeed())

		By("check if the connector's configmap is generated")
		var connectorConfigMap corev1.ConfigMap

		Eventually(func() bool {
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      v1beta1.GenConnectorConfigMapName(connector.Name, false) + "-for-" + connector.Spec.Streams[0],
				Namespace: connector.Namespace,
			}, &connectorConfigMap); err != nil {
				return false
			}

			return true
		})

		By("check if the connector's deployment is generated")
		var connectorDeployment appsv1.Deployment

		Eventually(func() bool {
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      v1beta1.GenConnectorDeploymentName(connector.Name, connector.Spec.Streams[0]),
				Namespace: connector.Namespace,
			}, &connectorDeployment); err != nil {
				return false
			}

			return true
		})
	})
})
