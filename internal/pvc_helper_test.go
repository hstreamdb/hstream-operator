package internal_test

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PvcHelper", func() {
	var hdb *appsv1alpha1.HStreamDB

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	Context("with volumeClaimTemplate not nil", func() {
		BeforeEach(func() {
			hdb.Spec.VolumeClaimTemplate = &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
			}
		})
		It("get pvc name", func() {
			Expect(internal.GetPvcName(hdb)).To(Equal(hdb.Name + "-" + hdb.Spec.VolumeClaimTemplate.Name))
		})
	})
	Context("with volumeClaimTemplate nil", func() {
		BeforeEach(func() {
			hdb.Spec.VolumeClaimTemplate = nil
		})
		It("get default pvc name", func() {
			Expect(internal.GetPvcName(hdb)).To(Equal(hdb.Name + "-" + "shard"))
		})
	})
})
