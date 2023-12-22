package internal

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetObjectMetadata returns the ObjectMetadata for a component
func GetObjectMetadata(hdb *hapi.HStreamDB, base *metav1.ObjectMeta, compType hapi.ComponentType) metav1.ObjectMeta {
	var metadata *metav1.ObjectMeta

	if base != nil {
		metadata = base.DeepCopy()
	} else {
		metadata = &metav1.ObjectMeta{}
	}
	metadata.Name = compType.GetResName(hdb)
	metadata.Namespace = hdb.Namespace

	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}

	metadata.Labels[hapi.InstanceKey] = hdb.Name
	metadata.Labels[hapi.ComponentKey] = string(compType)

	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	return *metadata
}
