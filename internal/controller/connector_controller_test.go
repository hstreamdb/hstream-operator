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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

var _ = Describe("controller/connector", Ordered, func() {
	connectorTpl := mock.CreateDefaultConnectorTemplate()
	connectorTpl.Namespace = "connector-test"

	BeforeAll(func() {
		Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: connectorTpl.Namespace,
			},
		})).Should(Succeed())

		By("creating a connector template")
		Expect(k8sClient.Create(context.TODO(), &connectorTpl)).Should(Succeed())
	})

	It("should create/delete a connector successfully", func() {
		connector := mock.CreateDefaultConnector("connector-test")
		connector.Name = connector.Name + "-1"
		var configMap corev1.ConfigMap
		var deployment appsv1.Deployment
		configMapName, deploymentName := getConnectorSubResourceName(&connector)

		By("creating a connector")
		Expect(k8sClient.Create(context.TODO(), &connector)).Should(Succeed())

		By("check if the connector's configmap is generated")
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      configMapName,
				Namespace: connector.Namespace,
			}, &configMap)
		}).Should(BeNil())

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
			Name:               "test-connector-1",
			UID:                connector.UID,
			Controller:         &[]bool{true}[0],
			BlockOwnerDeletion: &[]bool{true}[0],
		}

		By("check if the owner reference of the configmap is set")
		Expect(configMap.OwnerReferences).To(ContainElement(expectedOwnerReference))

		By("check if the owner reference of the deployment is set")
		Expect(deployment.OwnerReferences).To(ContainElement(expectedOwnerReference))
	})

	It("should set connector container ports and resources", func() {
		connector := mock.CreateDefaultConnector("connector-test")
		connector.Name = connector.Name + "-2"
		var deployment appsv1.Deployment
		_, deploymentName := getConnectorSubResourceName(&connector)

		connector.Spec.Container = corev1.Container{
			Ports: []corev1.ContainerPort{
				{
					Name:          "prom",
					ContainerPort: 9400,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		}

		By("creating a connector")
		Expect(k8sClient.Create(context.TODO(), &connector)).Should(Succeed())

		By("check if the connector's deployment is generated")
		Eventually(func() error {
			return k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      deploymentName,
				Namespace: connector.Namespace,
			}, &deployment)
		}).Should(BeNil())

		Expect(deployment.Spec.Template.Spec.Containers[0].Ports).To(ContainElement(connector.Spec.Container.Ports[0]))
		Expect(deployment.Spec.Template.Spec.Containers[0].Resources).To(Equal(connector.Spec.Container.Resources))
	})
})

func getConnectorSubResourceName(connector *v1beta1.Connector) (string, string) {
	return v1beta1.GenConnectorConfigMapNameForStream(connector.Name, connector.Spec.Streams[0]),
		v1beta1.GenConnectorDeploymentName(connector.Name, connector.Spec.Streams[0])
}
