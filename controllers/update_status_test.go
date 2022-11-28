package controllers

import (
	"context"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UpdateStatus", func() {
	var hdb *appsv1alpha1.HStreamDB
	updateStatus := updateStatus{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		var requeue *requeue
		BeforeEach(func() {
			hdb.Status.HStoreConfigured = true
			requeue = updateStatus.reconcile(ctx, clusterReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		When("hserver has been bootstrapped", func() {
			BeforeEach(func() {
				hdb.Status.HServerConfigured = true
				requeue = updateStatus.reconcile(ctx, clusterReconciler, hdb)
			})

			It("should not requeue", func() {
				Expect(requeue).To(BeNil())
			})

			It("should bootstrapped hserver", func() {
				newHdb := &appsv1alpha1.HStreamDB{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), newHdb)
				Expect(err).To(BeNil())
				Expect(newHdb.Status.HStoreConfigured).To(BeTrue())
				Expect(newHdb.Status.HServerConfigured).To(BeTrue())
			})
		})
	})
})
