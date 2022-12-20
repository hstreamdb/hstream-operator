package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// this test case requires to connect to the running k8s cluster in local or anywhere
// set env: export USE_EXISTING_CLUSTER=true
// run the test: ginkgo run --label-filter 'k8s'
var _ = Describe("BootstrapHServer", Label("k8s"), func() {
	timeout := 5 * time.Minute

	var hdb *appsv1alpha1.HStreamDB
	var requeue *requeue
	addServices := addServices{}
	addHStore := addHStore{}
	addHServer := addHServer{}
	addAdminServer := addAdminServer{}
	bootstrapHServer := bootstrapHServer{}
	bootstrapHStore := bootstrapHStore{}
	updateConfigMap := updateConfigMap{}
	ctx := context.TODO()

	BeforeEach(func() {
		if !isUsingExistingCluster() {
			Skip("Skip testcase BootstrapHServer")
		}

		hdb = mock.CreateDefaultCR()
		//hdb.Name = "bootstrap"
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
		Expect(requeue).To(BeNil())
		requeue = addServices.reconcile(ctx, clusterReconciler, hdb)
		Expect(requeue).To(BeNil())
		requeue = addHStore.reconcile(ctx, clusterReconciler, hdb)
		Expect(requeue).To(BeNil())
		requeue = addAdminServer.reconcile(ctx, clusterReconciler, hdb)
		Expect(requeue).To(BeNil())
	})

	JustBeforeEach(func() {
		Eventually(func() bool {
			sts := &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: appsv1alpha1.ComponentTypeHStore.GetResName(hdb.Name),
				},
			}
			if err := checkPodRunningStatus(ctx, k8sClient, hdb, sts); err != nil {
				By(fmt.Sprint("CheckPodRunningStatus failed. ", err.Error()))
				return false
			}
			return true
		}, timeout, 10*time.Second).Should(BeTrue())
	})

	AfterEach(func() {
		k8sClient.Delete(ctx, hdb)
	})

	Context("with bootstrap first", func() {
		It("bootstrap", func() {
			By("Bootstrap hstore")
			Eventually(func() bool {
				requeue = bootstrapHStore.reconcile(ctx, clusterReconciler, hdb)
				if requeue == nil || (requeue.curError == nil && requeue.message == "") {
					return true
				}
				return false
			}, timeout, 10*time.Second).Should(BeTrue())

			Expect(hdb.Status.HStoreConfigured).To(BeTrue())

			By("Add hserver")
			requeue = addHServer.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue).To(BeNil())

			By("Checking the hserver StatefulSet's ready replicas")
			Eventually(func() bool {
				sts := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: appsv1alpha1.ComponentTypeHServer.GetResName(hdb.Name),
					},
				}
				if err := checkPodRunningStatus(ctx, k8sClient, hdb, sts); err != nil {
					By(fmt.Sprint("CheckPodRunningStatus failed. ", err.Error()))
					return false
				}
				return true
			}, timeout, 10*time.Second).Should(BeTrue())

			By("Bootstrap hserver")
			Eventually(func() bool {
				requeue = bootstrapHServer.reconcile(ctx, clusterReconciler, hdb)
				if requeue == nil || (requeue.curError == nil && requeue.message == "") {
					return true
				}
				return false
			}, timeout, 10*time.Second).Should(BeTrue())

			Expect(hdb.Status.HServerConfigured).To(BeTrue())

			By("hServer has been bootstrapped")
			requeue = bootstrapHServer.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue).To(BeNil())

			By("hStroe has been bootstrapped")
			requeue = bootstrapHStore.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue).To(BeNil())
		})
	})
})
