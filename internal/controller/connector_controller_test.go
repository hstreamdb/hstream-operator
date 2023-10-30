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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

var _ = Describe("controller/connector", func() {
	connectorTpl := mock.CreateDefaultConnectorTemplate()
	connectorTpl.Namespace = "connector-test"
	connector := mock.CreateDefaultConnector()
	connector.Namespace = "connector-test"

	It("should create/delete a connector successfully", func() {
		Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: connector.Namespace,
			},
		})).Should(Succeed())

		By("creating a connector template")
		Expect(k8sClient.Create(context.TODO(), &connectorTpl)).Should(Succeed())

		By("creating a connector")
		Expect(k8sClient.Create(context.TODO(), &connector)).Should(Succeed())

		var configMap corev1.ConfigMap
		configMapName := v1beta1.GenConnectorConfigMapNameForStream(connector.Name, connector.Spec.Streams[0])

		By("check if the connector's configmap is generated")
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      configMapName,
				Namespace: connector.Namespace,
			}, &configMap)
		}).Should(BeNil())

		var deployment appsv1.Deployment
		deploymentName := v1beta1.GenConnectorDeploymentName(connector.Name, connector.Spec.Streams[0])

		By("check if the connector's deployment is generated")
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      deploymentName,
				Namespace: connector.Namespace,
			}, &deployment)
		}).Should(BeNil())

		expectedOwnerReference := metav1.OwnerReference{
			APIVersion:         "apps.hstream.io/v1beta1",
			Kind:               "Connector",
			Name:               "test-connector",
			UID:                connector.UID,
			Controller:         &[]bool{true}[0],
			BlockOwnerDeletion: &[]bool{true}[0],
		}

		By("check if the owner reference of the configmap is set")
		Expect(configMap.OwnerReferences).To(ContainElement(expectedOwnerReference))

		By("check if the owner reference of the deployment is set")
		Expect(deployment.OwnerReferences).To(ContainElement(expectedOwnerReference))
	})
})
