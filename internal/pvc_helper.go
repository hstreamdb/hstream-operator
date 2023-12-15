package internal

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPvc(hdb *hapi.HStreamDB, template *corev1.PersistentVolumeClaimTemplate, compType hapi.ComponentType) (pvc corev1.PersistentVolumeClaim) {
	var customMetadata *metav1.ObjectMeta

	if template != nil {
		pvc.ObjectMeta = *template.ObjectMeta.DeepCopy()
		pvc.Spec = *template.Spec.DeepCopy()
		customMetadata = &pvc.ObjectMeta
	}

	pvc.ObjectMeta = GetObjectMetadata(hdb, customMetadata, compType)
	pvc.ObjectMeta.Name = GetPvcName(hdb, template)

	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = corev1.ResourceList{}
	}

	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if (&storage).IsZero() {
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("128Gi")
	}
	return
}

func GetPvcName(hdb *hapi.HStreamDB, pvc *corev1.PersistentVolumeClaimTemplate) string {
	shortName := ""
	if pvc != nil && pvc.Name != "" {
		shortName = pvc.Name
	}
	return GetResNameWithDefault(hdb, shortName, "data")
}
