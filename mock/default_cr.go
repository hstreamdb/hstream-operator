package mock

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateDefaultCR() *hapi.HStreamDB {
	metadataReplication := int32(1)
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
				MetadataReplicateAcross: &metadataReplication,
				NShards:                 1,
				LogDeviceConfig: runtime.RawExtension{
					Raw: []byte("{}"),
				},
			},
			HServer: hapi.Component{
				Image:           "hstreamdb/hstream:rqlite",
				ImagePullPolicy: "IfNotPresent",
				Replicas:        1,
				Container:       hapi.Container{
					//Command: []string{
					//	"/usr/local/bin/hstream-server",
					//},
					//Args: []string{
					//	"--config-path",
					//	"/etc/hstream/config.yaml",
					//	"--bind-address",
					//	"0.0.0.0",
					//	"--advertised-address",
					//	"$(POD_IP)",
					//	"--port",
					//	"6570",
					//	"--internal-port",
					//	"6571",
					//	"--seed-nodes",
					//	"hstreamdb-sample-hserver-0.hstreamdb-sample-hserver:6571",
					//	"--server-id",
					//	"100",
					//	"--metastore-uri",
					//	"rq://rqlite-svc.default:4001",
					//	"--store-config",
					//	"/etc/logdevice/config.json",
					//	"--store-admin-host",
					//	"hstreamdb-sample-admin-server",
					//},
				},
			},
			HStore: hapi.Component{
				Image:               "hstreamdb/hstream:rqlite",
				ImagePullPolicy:     "IfNotPresent",
				Replicas:            3,
				VolumeClaimTemplate: nil,
			},
			AdminServer: hapi.Component{
				Image:           "hstreamdb/hstream:rqlite",
				ImagePullPolicy: "IfNotPresent",
				Replicas:        1,
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
