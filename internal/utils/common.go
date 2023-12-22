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

package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func MergeContainerPorts(dst []corev1.ContainerPort, src ...corev1.ContainerPort) []corev1.ContainerPort {
	// Create a map to track existing ports by name
	existingPorts := make(map[string]int)

	// Initialize the merged array
	var mergedPorts []corev1.ContainerPort

	// Add ports from the first set
	for _, port := range dst {
		existingPorts[port.Name] = len(mergedPorts)
		mergedPorts = append(mergedPorts, port)
	}

	// Add or override ports from the second set
	for _, port := range src {
		if index, exists := existingPorts[port.Name]; exists {
			// Override the existing port with the new one
			mergedPorts[index] = port
		} else {
			// Add the new port
			existingPorts[port.Name] = len(mergedPorts)
			mergedPorts = append(mergedPorts, port)
		}
	}

	return mergedPorts
}

func OverrideContainerPorts(ports []corev1.ContainerPort, name, port string) []corev1.ContainerPort {
	newPorts := make([]corev1.ContainerPort, len(ports))
	copy(newPorts, ports)

	for i := range newPorts {
		portName := newPorts[i].Name

		if portName == name {
			newPorts[i].ContainerPort = intstr.Parse(port).IntVal
		}
	}

	return newPorts
}

func FindContainerPortByName(ports []corev1.ContainerPort, name string) *corev1.ContainerPort {
	for _, port := range ports {
		if port.Name == name {
			return &port
		}
	}

	return nil
}
