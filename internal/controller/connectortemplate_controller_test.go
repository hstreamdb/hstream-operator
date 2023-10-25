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

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("controller/connectortemplate", func() {
	connectorTpl := mock.CreateDefaultConnectorTemplate()

	Context("reconcile", func() {
		It("should create a connector template successfully", func() {
			fakeClient := fake.NewClientBuilder().WithRuntimeObjects(&connectorTpl).Build()
			reconciler := ConnectorTemplateReconciler{
				Client: fakeClient,
				Scheme: scheme.Scheme,
			}

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      connectorTpl.Name,
					Namespace: connectorTpl.Namespace,
				}},
			)

			Expect(err).ShouldNot(HaveOccurred())

			By("check if the connector template's configmap is generated")
			var configMap corev1.ConfigMap
			err = fakeClient.Get(context.TODO(), types.NamespacedName{
				Name:      v1beta1.GenConnectorConfigMapName(connectorTpl.Name, true),
				Namespace: connectorTpl.Namespace,
			}, &configMap)

			Expect(err).ShouldNot(HaveOccurred())

			By("delete the connector template")
			err = fakeClient.Delete(context.TODO(), &connectorTpl)

			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
