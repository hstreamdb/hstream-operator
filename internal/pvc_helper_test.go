package internal_test

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PvcHelper", func() {
	var hdb *hapi.HStreamDB

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	Context("with volumeClaimTemplate not nil", func() {
		var pvc corev1.PersistentVolumeClaim
		BeforeEach(func() {
			hdb.Spec.HStore.VolumeClaimTemplate = &corev1.PersistentVolumeClaimTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
			}
		})
		It("get pvc", func() {
			pvc = internal.GetPvc(hdb, hdb.Spec.HStore.VolumeClaimTemplate, hapi.ComponentTypeHStore)
			Expect(internal.GetPvcName(hdb, hdb.Spec.HStore.VolumeClaimTemplate)).To(
				Equal(hdb.Name + "-" + hdb.Spec.HStore.VolumeClaimTemplate.Name))
		})

		It("pvc name should be hdbName-data", func() {
			Expect(pvc.Name).To(Equal(internal.GetResNameOnPanic(hdb, "data")))
		})

		It("access mode should be ReadWriteOnce", func() {
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("resources.Requests should not be nil", func() {
			Expect(pvc.Spec.Resources.Requests).NotTo(BeNil())
		})

		It("should has default ResourceStorage", func() {
			Expect(pvc.Spec.Resources.Requests).To(HaveKeyWithValue(corev1.ResourceStorage, resource.MustParse("128Gi")))
		})
	})

	Context("with volumeClaimTemplate nil", func() {
		BeforeEach(func() {
			hdb.Spec.HStore.VolumeClaimTemplate = nil
		})
		It("get default pvc name", func() {
			Expect(internal.GetPvcName(hdb, hdb.Spec.HStore.VolumeClaimTemplate)).To(
				Equal(hdb.Name + "-" + "data"))
		})
	})

	Context("with get volume", func() {
		var volume corev1.Volume
		var m internal.ConfigMap
		BeforeEach(func() {
			m = internal.ConfigMap{
				MountName:     "shard-path",
				MapNameSuffix: "shard",
				MapKey:        "config.json",
				MapPath:       "config.json",
			}
			volume = internal.GetVolume(hdb, m)
		})

		It("get volume from configMap", func() {
			Expect(volume.Name).To(Equal(m.MountName))
		})

		It("volumeSource should be configMap", func() {
			Expect(volume.VolumeSource.ConfigMap).NotTo(BeNil())
		})

		It("configMap name should be hdbName-MapNameSuffix", func() {
			Expect(volume.VolumeSource.ConfigMap.Name).To(Equal(internal.GetResNameOnPanic(hdb, m.MapNameSuffix)))
		})

		It("should has element that name same as MapKey", func() {
			Expect(volume.VolumeSource.ConfigMap.Items).To(ContainElement(
				corev1.KeyToPath{Key: m.MapKey, Path: m.MapPath},
			))
		})
	})
})
