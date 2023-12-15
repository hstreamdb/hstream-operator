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
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=hdb
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.gateway.endpoint"
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
	// ExternalHMeta set external HMeta cluster addr
	// +optional
	ExternalHMeta *ExternalHMeta `json:"externalHmeta,omitempty"`

	Config Config `json:"config,omitempty"`

	Gateway     *Gateway   `json:"gateway,omitempty"`
	Console     *Component `json:"console,omitempty"`
	AdminServer Component  `json:"adminServer,omitempty"`
	HServer     Component  `json:"hserver,omitempty"`
	HStore      Component  `json:"hstore,omitempty"`
	HMeta       Component  `json:"hmeta,omitempty"`
}

type ExternalHMeta struct {
	// Host set external HMeta cluster host, it can be ip addr or service name
	// +kubebuilder:validation:Required
	Host string `json:"host"`
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
	// Namespace the namespace of external HMeta cluster
	// +kubebuilder:default:=default
	// +optional
	Namespace string `json:"namespace"`
}

type Config struct {
	// MetadataReplicateAcross metadata replication must less than or equal to HStore replicas.
	// If this is not specified, it will be set to HStore replicas or 3 if HStore replica more than 3
	// Cannot be updated.
	// More info: https://logdevice.io/docs/Config.html#metadata-logs-metadata-logs
	//
	// +kubebuilder:validation:Minimum:=1
	// +optional
	MetadataReplicateAcross *int32 `json:"metadata-replicate-across,omitempty"`

	// NShards the number of HStore data shard
	// Cannot be updated.
	//
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=1
	// +optional
	NShards int32 `json:"nshards,omitempty"`

	// log device bootstrap config, json style
	// More info: https://logdevice.io/docs/Config.html
	// Example: https://github.com/hstreamdb/hstream/blob/main/deploy/k8s/config.json
	//
	// +optional
	LogDeviceConfig runtime.RawExtension `json:"logDeviceConfig,omitempty"`
}

// HStreamDBStatus defines the observed state of HStreamDB
type HStreamDBStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// HMeta store the status of HMeta cluster
	HMeta HMetaStatus `json:"hmeta"`
}

type HMetaStatus struct {
	// Nodes the status of node that return by api http://localhost:4001/status?pretty in HMeta pod
	Nodes   []HMetaNode `json:"nodes"`
	Version string      `json:"version"`
}

type HMetaNode struct {
	NodeId    string `json:"nodeId"`
	Reachable bool   `json:"reachable"`
	Leader    bool   `json:"leader"`
	Error     string `json:"error,omitempty"`
}

func init() {
	SchemeBuilder.Register(&HStreamDB{}, &HStreamDBList{})
}

func (er *ExternalHMeta) GetAddr() string {
	addr := er.Host
	if er.Namespace != "" {
		addr += "." + er.Namespace
	}
	return addr + ":" + strconv.Itoa(int(er.Port))
}
