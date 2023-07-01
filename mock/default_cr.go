package mock

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateDefaultCR() *hapi.HStreamDB {
	return &hapi.HStreamDB{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HStreamDB",
			APIVersion: "apps.hstream.io/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hstreamdb-sample",
			Namespace: "default",
		},
		Spec: hapi.HStreamDBSpec{
			Config: hapi.Config{
				MetadataReplicateAcross: &[]int32{1}[0],
				NShards:                 1,
				LogDeviceConfig: runtime.RawExtension{
					Raw: []byte("{}"),
				},
			},
			AdminServer: hapi.Component{
				Image:           "hstreamdb/hstream:rqlite",
				ImagePullPolicy: "IfNotPresent",
				Replicas:        1,
			},
			HServer: hapi.Component{
				Image:           "hstreamdb/hstream:rqlite",
				ImagePullPolicy: "IfNotPresent",
				Replicas:        3,
			},
			HStore: hapi.Component{
				Image:               "hstreamdb/hstream:rqlite",
				ImagePullPolicy:     "IfNotPresent",
				Replicas:            3,
				VolumeClaimTemplate: nil,
			},
			HMeta: hapi.Component{
				Image:               "rqlite/rqlite:latest",
				ImagePullPolicy:     "IfNotPresent",
				Replicas:            1,
				VolumeClaimTemplate: nil,
			},
		},
		Status: hapi.HStreamDBStatus{},
	}
}
