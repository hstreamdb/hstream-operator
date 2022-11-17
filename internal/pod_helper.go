package internal

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetObjectMetadata returns the ObjectMetadata for a component
func GetObjectMetadata(hdb *appsv1alpha1.HStreamDB, base *metav1.ObjectMeta, compType appsv1alpha1.ComponentType) metav1.ObjectMeta {
	var metadata *metav1.ObjectMeta

	if base != nil {
		metadata = base.DeepCopy()
	} else {
		metadata = &metav1.ObjectMeta{}
	}
	metadata.Namespace = hdb.Namespace

	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}

	metadata.Labels[appsv1alpha1.ComponentKey] = string(compType)

	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	return *metadata
}
