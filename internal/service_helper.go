package internal

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// GetService builds a service for a cluster.
func GetService(hdb *appsv1alpha1.HStreamDB, ports []corev1.ServicePort, compType appsv1alpha1.ComponentType) corev1.Service {
	service := corev1.Service{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
	}
	service.ObjectMeta.Name = GetResNameOnPanic(hdb, string(compType))
	service.Spec.Ports = make([]corev1.ServicePort, len(ports))
	for i := range ports {
		service.Spec.Ports[i] = *ports[i].DeepCopy()
	}
	service.Spec.Selector = map[string]string{
		appsv1alpha1.ComponentKey: string(compType),
	}
	return service
}

// GetHeadlessService builds a headless service for a cluster.
func GetHeadlessService(hdb *appsv1alpha1.HStreamDB, compType appsv1alpha1.ComponentType) corev1.Service {
	service := corev1.Service{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
	}
	service.ObjectMeta.Name = GetResNameOnPanic(hdb, string(compType))
	service.Spec.ClusterIP = corev1.ClusterIPNone
	service.Spec.Selector = map[string]string{
		appsv1alpha1.ComponentKey: string(compType),
	}
	return service
}
