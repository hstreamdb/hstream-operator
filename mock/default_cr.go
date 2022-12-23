package mock

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func CreateDefaultCR() *appsv1alpha1.HStreamDB {
	nShards := int32(1)
	replica := int32(1)
	hStoreReplica := int32(3)
	return &appsv1alpha1.HStreamDB{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HStreamDB",
			APIVersion: "apps.hstream.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hstreamdb-sample",
			Namespace: "default",
		},
		Spec: appsv1alpha1.HStreamDBSpec{
			Config: appsv1alpha1.Config{
				MetadataReplicateAcross: 1,
				NShards:                 &nShards,
				LogDeviceConfig: runtime.RawExtension{
					Raw: []byte("{}"),
				},
			},
			VolumeClaimTemplate: nil,
			Image:               "hstreamdb/hstream:rqlite",
			ImagePullPolicy:     "IfNotPresent",
			HServer: appsv1alpha1.Component{
				Replicas: &replica,
				Container: appsv1alpha1.Container{
					Command: []string{
						"/usr/local/bin/hstream-server",
					},
					Args: []string{
						"--config-path",
						"/etc/hstream/config.yaml",
						"--bind-address",
						"0.0.0.0",
						"--advertised-address",
						"$(POD_IP)",
						"--port",
						"6570",
						"--internal-port",
						"6571",
						"--seed-nodes",
						"hstreamdb-sample-hserver-0.hstreamdb-sample-hserver:6571",
						"--server-id",
						"100",
						"--metastore-uri",
						"rq://rqlite-svc.default:4001",
						"--store-config",
						"/etc/logdevice/config.json",
						"--store-admin-host",
						"hstreamdb-sample-admin-server",
					},
				},
			},
			HStore: appsv1alpha1.Component{
				Replicas: &hStoreReplica,
			},
			AdminServer: appsv1alpha1.Component{
				Replicas: &replica,
			},
		},
		Status: appsv1alpha1.HStreamDBStatus{},
	}
}
