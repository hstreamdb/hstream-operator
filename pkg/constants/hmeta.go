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

package constants

import (
	corev1 "k8s.io/api/core/v1"
)

// Check https://github.com/rqlite/kubernetes-configuration/blob/master/statefulset-3-node.yaml as an example.
var DefaultHMetaArgs = []string{
	"--disco-mode", "dns",
	"--join-interval", "1s",
	"--join-attempts", "120",
}

var DefaultHMetaPort = corev1.ContainerPort{
	Name:          "rqlite",
	ContainerPort: 4001,
	Protocol:      corev1.ProtocolTCP,
}
