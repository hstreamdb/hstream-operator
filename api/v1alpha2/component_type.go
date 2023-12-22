package v1alpha2

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComponentType string

const (
	ComponentTypeGateway     ComponentType = "gateway"
	ComponentTypeAdminServer ComponentType = "admin-server"
	ComponentTypeConsole     ComponentType = "console"
	ComponentTypeHServer     ComponentType = "hserver"
	ComponentTypeHStore      ComponentType = "hstore"
	ComponentTypeHMeta       ComponentType = "hmeta"
)

func (ct ComponentType) GetResName(instance string) string {
	return fmt.Sprintf("%s-%s", instance, ct)
}

func (ct ComponentType) GetObjectMeta(hdb *HStreamDB, meta *metav1.ObjectMeta) metav1.ObjectMeta {
	fixedMeta := metav1.ObjectMeta{
		Name:      ct.GetResName(hdb.Name),
		Namespace: hdb.Namespace,
		Labels: map[string]string{
			InstanceKey:  hdb.Name,
			ComponentKey: string(ct),
		},
	}

	if meta != nil {
		fixedMeta.Labels = mergeMap(fixedMeta.Labels, meta.Labels)
		fixedMeta.Annotations = mergeMap(fixedMeta.Annotations, meta.Annotations)
	}

	return fixedMeta
}

func (ct ComponentType) GetService(hdb *HStreamDB, ports []corev1.ServicePort, isInternal bool) corev1.Service {
	meta := ct.GetObjectMeta(hdb, nil)

	if isInternal {
		meta.Name = ct.GetResName(hdb.Name + "-internal")
	}

	return corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				ComponentKey: string(ct),
			},
			Ports: ports,
		},
	}
}

func (ct ComponentType) GetHeadlessService(hdb *HStreamDB, ports []corev1.ServicePort) corev1.Service {
	svc := ct.GetService(hdb, ports, true)
	svc.Spec.ClusterIP = corev1.ClusterIPNone
	svc.Spec.PublishNotReadyAddresses = true

	return svc
}

func mergeMap(existing, newMap map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy existing map
	for key, value := range existing {
		result[key] = value
	}

	// Merge with new map
	for key, value := range newMap {
		result[key] = value
	}

	return result
}
