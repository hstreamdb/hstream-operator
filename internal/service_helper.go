package internal

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
)

// GetService builds a service for a cluster.
func GetService(hdb *hapi.HStreamDB, compType hapi.ComponentType, ports ...corev1.ServicePort) corev1.Service {
	service := corev1.Service{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
	}
	service.Spec.Ports = make([]corev1.ServicePort, len(ports))
	for i := range ports {
		service.Spec.Ports[i] = *ports[i].DeepCopy()
	}
	service.Spec.Selector = map[string]string{
		hapi.ComponentKey: string(compType),
	}
	return service
}

// GetHeadlessService builds a headless service for a cluster.
func GetHeadlessService(hdb *hapi.HStreamDB, compType hapi.ComponentType, ports ...corev1.ServicePort) corev1.Service {
	service := GetService(hdb, compType, ports...)
	service.Name = GetResNameOnPanic(hdb, "internal-"+string(compType))
	service.Spec.ClusterIP = corev1.ClusterIPNone
	service.Spec.PublishNotReadyAddresses = true
	return service
}
