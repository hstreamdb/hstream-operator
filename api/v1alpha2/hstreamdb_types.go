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

package v1alpha2

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=hdb
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
	Config Config `json:"config,omitempty"`

	// +kubebuilder:validation:Required
	Rqlite Rqlite `json:"rqlite"`

	AdminServer Component `json:"adminServer,omitempty"`
	HServer     Component `json:"hserver,omitempty"`
	HStore      Component `json:"hstore,omitempty"`

	Console *Component `json:"console,omitempty"`
}

type Rqlite struct {
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// +kubebuilder:validation:Required
	Port int32 `json:"port"`

	Namespace string `json:"namespace"`
}

func (r *Rqlite) GetAddr() string {
	return fmt.Sprintf("%s.%s:%d", r.Host, r.Namespace, r.Port)
}

type Config struct {
	// KafkaMode decides whether to use Kafka protocol or not.
	//
	// +kubebuilder:default:=false
	// +optional
	KafkaMode bool `json:"kafkaMode,omitempty"`

	// Metrics decides whether to enable metrics or not.
	// Only available in KafkaMode.
	//
	// +kubebuilder:default:=false
	// +optional
	Metrics bool `json:"metrics,omitempty"`

	// +optional
	ExternalAccess *ExternalAccess `json:"externalAccess,omitempty"`

	// LogDevice config. Refer to https://logdevice.io/docs/Config.html for more details.
	//
	// +optional
	LogDeviceConfig runtime.RawExtension `json:"logDeviceConfig,omitempty"`

	// MetadataReplicateAcross decides the required metadata replication in LogDevice.
	// Default same as HStore replicas. If replicas is more than 3, it will be set to 3.
	//
	// Cannot be updated.
	//
	// More info: https://logdevice.io/docs/Config.html#metadata-logs-metadata-logs
	//
	// +kubebuilder:validation:Minimum:=1
	// +optional
	MetadataReplicateAcross *int32 `json:"metadata-replicate-across,omitempty"`

	// NShards decides the number of shards per LogDevice daemon.
	//
	// Cannot be updated.
	//
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=1
	// +optional
	NShards int32 `json:"nshards,omitempty"`
}

type AccessType string

const (
	AccessTypeNodePort AccessType = "nodeport"
)

type ExternalAccess struct {
	// +kube:validation:Required
	AccessType AccessType `json:"accessType"`

	AdvertisedAddresses []AdvertisedAddress `json:"advertisedAddresses"`
}

type AdvertisedAddress struct {
	Instance uint `json:"instance"`

	Host string `json:"host"`

	Port int32 `json:"port"`

	NodePort int32 `json:"nodePort"`
}

// HStreamDBStatus defines the observed state of HStreamDB
type HStreamDBStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	InternalRqlite InternalRqliteStatus `json:"internalRqlite,omitempty"`
}

type InternalRqliteStatus struct {
	Nodes map[string]RqliteNode `json:"nodes"`
}

type RqliteNode struct {
	Id        string `json:"id"`
	Addr      string `json:"addr"`
	Reachable bool   `json:"reachable"`
	Leader    bool   `json:"leader"`
}

func (rs *InternalRqliteStatus) IsAllNodesReady() bool {
	if len(rs.Nodes) == 0 {
		return false
	}

	for _, node := range rs.Nodes {
		if !node.Reachable {
			return false
		}
	}

	return true
}

func init() {
	SchemeBuilder.Register(&HStreamDB{}, &HStreamDBList{})
}
