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

package v1beta1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
)

var _ = Describe("v1beta1/connector_utils", func() {
	It("should generate correct configmap name", func() {
		connectorTplName := "test-connector-tpl"
		connectorName := "test-connector"

		configMap1 := v1beta1.GenConnectorConfigMapName(connectorTplName, true)
		Expect(configMap1).To(Equal("test-connector-tpl-hct"))

		configMap2 := v1beta1.GenConnectorConfigMapName(connectorName, false)
		Expect(configMap2).To(Equal("test-connector-hc"))
	})

	It("should generate correct deployment name", func() {
		connectorName := "test-connector"
		stream := "test-stream"

		deploymentName := v1beta1.GenConnectorDeploymentName(connectorName, stream)
		Expect(deploymentName).To(Equal("test-connector-test-stream-hc"))
	})
})
