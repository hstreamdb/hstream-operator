package controllers

import (
	"context"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AddHstore", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	hStore := addHStore{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		BeforeEach(func() {
			requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
		})
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var sts *appsv1.StatefulSet
		It("should successfully get sts", func() {
			var err error
			sts, err = getHStoreStatefulSet(hdb)
			Expect(err).To(BeNil())
		})

		When("hserver has been deploy", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newSts, err := getHStoreStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.UID).To(Equal(newSts.UID))
				})
			})

			Context("update container name", func() {
				name := "hdb-hserver"
				BeforeEach(func() {
					hdb.Spec.HStore.Container.Name = name
					requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container name", func() {
					deploy, err := getHStoreStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
				})
			})
			Context("update container command", func() {
				command := []string{"bash", "-c", "|", "echo 'hello world'"}
				BeforeEach(func() {
					hdb.Spec.HStore.Container.Command = command
					requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container command", func() {
					sts, err := getHStoreStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(sts.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
				})
			})
		})
	})

	Context("with a reconciled cluster that used pvc", func() {
		storageClassName := "standard"
		BeforeEach(func() {
			sts, err := getHStoreStatefulSet(hdb)
			if err != nil {
				Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
			} else {
				_ = k8sClient.Delete(ctx, sts)
			}

			hdb.Spec.HStore.VolumeClaimTemplate = &corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		It("should get new sts with pvc", func() {
			sts, err := getHStoreStatefulSet(hdb)
			Expect(err).To(BeNil())
			Expect(sts.Spec.VolumeClaimTemplates).To(HaveLen(1))
			Expect(*sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(Equal(storageClassName))
			Expect(sts.Spec.VolumeClaimTemplates[0].Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))

		})
	})
})

func getHStoreStatefulSet(hdb *hapi.HStreamDB) (sts *appsv1.StatefulSet, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHStore.GetResName(hdb.Name),
	}
	sts = &appsv1.StatefulSet{}
	err = k8sClient.Get(context.TODO(), keyObj, sts)
	return
}
