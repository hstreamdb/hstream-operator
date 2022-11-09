/*
Copyright 2022.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=hdb

// HStreamDB is the Schema for the hstreamdbs API
type HStreamDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HStreamDBSpec   `json:"spec,omitempty"`
	Status HStreamDBStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HStreamDBList contains a list of HStreamDB
type HStreamDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HStreamDB `json:"items"`
}

// HStreamDBSpec defines the desired state of HStreamDB
type HStreamDBSpec struct {
	// +optional
	Config Config `json:"config,omitempty"`

	// VolumeClaimTemplate allows customizing the persistent volume claim for the
	// pod.
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`

	Image string `json:"image,omitempty"`

	HServer     Component `json:"hserver,omitempty"`
	HStore      Component `json:"hstore,omitempty"`
	AdminServer Component `json:"adminServer,omitempty"`
}

// HStreamDBStatus defines the observed state of HStreamDB
type HStreamDBStatus struct {
	Configured bool `json:"configured"`
}

type Config struct {
	//+kubebuilder:default:=1
	//+kubebuilder:validation:Minimum:=1
	MetadataReplicateAcross int32 `json:"metadata-replicate-across,omitempty"`
	//+kubebuilder:default:=1
	//+kubebuilder:validation:Minimum:=1
	NShards *int32 `json:"nshards,omitempty"`
	// log device bootstrap config, json style
	// More info: https://logdevice.io/docs/Settings.html
	// Example: https://github.com/hstreamdb/hstream/blob/main/deploy/k8s/config.json
	// +kubebuilder:pruning:PreserveUnknownFields
	LogDeviceConfig runtime.RawExtension `json:"LogDeviceConfig,omitempty"`
}

func init() {
	SchemeBuilder.Register(&HStreamDB{}, &HStreamDBList{})
}
