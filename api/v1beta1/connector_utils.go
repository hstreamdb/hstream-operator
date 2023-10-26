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

import "strings"

func GenConnectorConfigMapName(connectorName string, isTemplate bool) (suffix string) {
	suffix += connectorName + "-h" // Short for hstreamio.

	if isTemplate {
		suffix += "ct" // Short for connector template.
	} else {
		suffix += "c" // Short for connector.
	}

	return
}

func GenConnectorConfigMapNameForStream(connectorName, stream string) string {
	return GenConnectorConfigMapName(connectorName, false) + "-for-" + strings.Replace(stream, "_", "-", -1)
}

func GenConnectorDeploymentName(connectorName, stream string) string {
	return connectorName + "-" + strings.Replace(stream, "_", "-", -1) + "-hc"
}
