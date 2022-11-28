package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AddHstore", func() {
	var hdb *appsv1alpha1.HStreamDB
	var requeue *requeue
	hStore := addHStore{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = hStore.reconcile(ctx, clusterReconciler, hdb)
	})

	AfterEach(func() {
		k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var sts *appsv1.StatefulSet
		It("should successfully get sts", func() {
			var err error
			sts, err = getHStoreStatefulSet(hdb)
			Expect(err).To(BeNil())
		})

		It("should have default port in args", func() {
			Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
			Expect(sts.Spec.Template.Spec.Containers[0].Args).Should(ContainElements(
				"--port", "4440",
				"--gossip-port", "4441",
				"--admin-port", "6440",
				"--num-shards", fmt.Sprint(*hdb.Spec.Config.NShards)))
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
})

func getHStoreStatefulSet(hdb *appsv1alpha1.HStreamDB) (sts *appsv1.StatefulSet, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      appsv1alpha1.ComponentTypeHStore.GetResName(hdb.Name),
	}
	sts = &appsv1.StatefulSet{}
	err = k8sClient.Get(context.TODO(), keyObj, sts)
	return
}
