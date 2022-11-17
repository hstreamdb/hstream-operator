package internal

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPvc(hdb *appsv1alpha1.HStreamDB) (pvc corev1.PersistentVolumeClaim) {
	if hdb.Spec.VolumeClaimTemplate != nil {
		pvc = *hdb.Spec.VolumeClaimTemplate.DeepCopy()
	}

	pvc.ObjectMeta = getPvcMetadata(hdb, appsv1alpha1.ComponentTypeHStore)
	pvc.ObjectMeta.Name = GetPvcName(hdb)

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

func GetPvcName(hdb *appsv1alpha1.HStreamDB) string {
	shortName := ""
	if hdb.Spec.VolumeClaimTemplate != nil && hdb.Spec.VolumeClaimTemplate.Name != "" {
		shortName = hdb.Spec.VolumeClaimTemplate.Name
	}
	return GetResNameWithDefault(hdb, shortName, "shard")
}

func GetVolume(hdb *appsv1alpha1.HStreamDB, m ConfigMap) corev1.Volume {
	return corev1.Volume{
		Name: m.MountName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: GetResNameOnPanic(hdb, m.MapNameSuffix)},
				Items: []corev1.KeyToPath{
					{Key: m.MapKey, Path: m.MapPath},
				},
			},
		},
	}
}

// getPvcMetadata returns the metadata for a PVC
func getPvcMetadata(hdb *appsv1alpha1.HStreamDB, compType appsv1alpha1.ComponentType) metav1.ObjectMeta {
	var customMetadata *metav1.ObjectMeta

	if hdb.Spec.VolumeClaimTemplate != nil {
		customMetadata = &hdb.Spec.VolumeClaimTemplate.ObjectMeta
	}
	return GetObjectMetadata(hdb, customMetadata, compType)
}
