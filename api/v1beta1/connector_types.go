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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConnectorSpec defines the desired state of Connector
type ConnectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type is the type of the connector, typically used to verify that the type matches the configuration.
	//
	// Each connector type is associated with a connector image, which is used to create the connector container.
	// View `ConnectorImageMap` for more details.
	// +kubebuilder:validation:Enum=sink-elasticsearch
	// +kubebuilder:validation:Required
	Type ConnectorType `json:"type"`

	// TemplateName is the name of the connector template.
	// +kubebuilder:validation:Required
	TemplateName string `json:"templateName"`

	// Streams is used to specify the streams that the connector will be applied to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Streams []string `json:"streams"`

	// Patches is used to specify the patches that will be applied to the connector configuration.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Patches json.RawMessage `json:"patches,omitempty"`

	// HServerEndpoint is the endpoint of the HStreamDB server.
	// For example: "hstreamdb-hserver:6570"
	// +kube:validation:Required
	HServerEndpoint string `json:"hserverEndpoint"`

	// ImageRegistry is used to specify the registry of the connector container image.
	// +optional
	ImageRegistry *string `json:"imageRegistry,omitempty"`

	// Deprecated: use `Container.Ports` instead.
	// ContainerPorts is used to specify the ports that will be exposed by the connector container.
	// +optional
	ContainerPorts []corev1.ContainerPort `json:"containerPorts,omitempty"`

	// Container is used to override the default connector container fields.
	// Note that not all fields are supported.
	// +optional
	Container corev1.Container `json:"container,omitempty"`

	// Containers is used to add additional containers to the connector pod.
	// +optional
	Containers []corev1.Container `json:"containers,omitempty"`
}

// ConnectorStatus defines the observed state of Connector
type ConnectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Connector is the Schema for the connectors API
type Connector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectorSpec   `json:"spec,omitempty"`
	Status ConnectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConnectorList contains a list of Connector
type ConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Connector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Connector{}, &ConnectorList{})
}
