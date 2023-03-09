package controllers

import (
	"context"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UpdateStatus", func() {
	var hdb *hapi.HStreamDB
	updateStatus := updateStatus{}
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
		var requeue *requeue
		BeforeEach(func() {
			hdb.Status.HMeta.Nodes = []hapi.HMetaNode{
				{
					NodeId:    "node-id",
					Reachable: true,
					Leader:    false,
					Error:     "",
				},
			}
			requeue = updateStatus.reconcile(ctx, clusterReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		When("hmeta has been ready", func() {
			BeforeEach(func() {
				hdb.Status.HStore.Bootstrapped = true
				requeue = updateStatus.reconcile(ctx, clusterReconciler, hdb)
			})

			It("should not requeue", func() {
				Expect(requeue).To(BeNil())
			})

			When("hserver has been bootstrapped", func() {
				BeforeEach(func() {
					hdb.Status.HServer.Bootstrapped = true
					requeue = updateStatus.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should bootstrapped hserver", func() {
					newHdb := &hapi.HStreamDB{}
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), newHdb)
					Expect(err).To(BeNil())
					Expect(newHdb.Status.HStore.Bootstrapped).To(BeTrue())
					Expect(newHdb.Status.HServer.Bootstrapped).To(BeTrue())
				})
			})
		})

	})
})
