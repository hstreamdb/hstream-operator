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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConnectorTemplateSpec defines the desired state of ConnectorTemplate
type ConnectorTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type is the type of the connector template, typically used to verify that the type matches the configuration.
	// +kubebuilder:validation:Required
	Type ConnectorType `json:"type"`

	// Config is the configuration for the connector template.
	// For details, see https://docs.hstream.io/ingest-and-distribute/connectors.html.
	//
	// Note: currently, only the `sink-elasticsearch` connector is supported.
	// +kubebuilder:validation:Required
	Config string `json:"config"`
}

// ConnectorTemplateStatus defines the observed state of ConnectorTemplate
type ConnectorTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ConnectorTemplate is the Schema for the connectortemplates API
type ConnectorTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectorTemplateSpec   `json:"spec,omitempty"`
	Status ConnectorTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConnectorTemplateList contains a list of ConnectorTemplate
type ConnectorTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConnectorTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConnectorTemplate{}, &ConnectorTemplateList{})
}
