package controllers

import (
	"context"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		// Delete all components to prevent contamination of other use cases
		_ = k8sClient.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace(hdb.Namespace))
		_ = k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(hdb.Namespace))
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
	Context("check conditions", Label("conditions"), func() {
		When("just hStore has been ready", func() {
			JustBeforeEach(func() {
				prepareHStoreReady(ctx, hdb)
			})

			It("check conditions", func() {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
				Expect(hdb.Status.Conditions).To(ContainElements(
					And(
						HaveField("Type", Equal(hapi.HStoreReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.Ready)),
						HaveField("Status", Equal(metav1.ConditionFalse)),
					),
				))
			})
		})
		When("just hServer has been ready", func() {
			JustBeforeEach(func() {
				prepareHServerReady(ctx, hdb)
			})

			It("check conditions", func() {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
				Expect(hdb.Status.Conditions).To(ContainElements(
					And(
						HaveField("Type", Equal(hapi.HServerReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.Ready)),
						HaveField("Status", Equal(metav1.ConditionFalse)),
					),
				))
			})
		})
		When("just gateway has been ready", func() {
			JustBeforeEach(func() {
				prepareGatewayReady(ctx, hdb)
			})

			It("check conditions", func() {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
				Expect(hdb.Status.Conditions).To(ContainElements(
					And(
						HaveField("Type", Equal(hapi.GatewayReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.Ready)),
						HaveField("Status", Equal(metav1.ConditionFalse)),
					),
				))
			})
		})

		When("all components have been ready", func() {
			JustBeforeEach(func() {
				prepareHStoreReady(ctx, hdb)
				prepareHServerReady(ctx, hdb)
				prepareGatewayReady(ctx, hdb)
			})

			It("check conditions", func() {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
				Expect(hdb.Status.Conditions).To(ConsistOf(
					And(
						HaveField("Type", Equal(hapi.HStoreReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.HServerReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.GatewayReady)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(hapi.Ready)),
						HaveField("Status", Equal(metav1.ConditionTrue)),
					),
				))
			})
		})
	})
})

func prepareHStoreReady(ctx context.Context, hdb *hapi.HStreamDB) {
	Expect(addHStore{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())

	hStore := &appsv1.StatefulSet{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHStore),
	}
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hStore), hStore)).To(BeNil())
	hStore.Status.ObservedGeneration = hStore.Generation
	hStore.Status.Replicas = 1
	hStore.Status.ReadyReplicas = 1
	Expect(k8sClient.Status().Update(ctx, hStore)).To(BeNil())

	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
	hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
	hdb.Status.HStore.Bootstrapped = true
	Expect(updateStatus{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())
}

func prepareHServerReady(ctx context.Context, hdb *hapi.HStreamDB) {
	Expect(addHServer{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())

	hServer := &appsv1.StatefulSet{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHServer),
	}
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hServer), hServer)).To(BeNil())
	hServer.Status.ObservedGeneration = hServer.Generation
	hServer.Status.Replicas = 1
	hServer.Status.ReadyReplicas = 1
	Expect(k8sClient.Status().Update(ctx, hServer)).To(BeNil())

	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
	hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
	hdb.Status.HServer.Bootstrapped = true
	Expect(updateStatus{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())
}

func prepareGatewayReady(ctx context.Context, hdb *hapi.HStreamDB) {
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).To(BeNil())
	hdb.Spec.Gateway = &hapi.Gateway{}
	hdb.Spec.Gateway.Endpoint = "fake"
	hdb.Spec.Gateway.Port = 14789
	hdb.Spec.Gateway.Image = "hstreamdb/hstream-gateway:latest"
	hdb.Spec.Gateway.Replicas = 1
	hdb.Status.HServer.Bootstrapped = true
	Expect(addGateway{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())

	gateway := &appsv1.Deployment{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeGateway),
	}
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(gateway), gateway)).To(BeNil())
	gateway.Status.ObservedGeneration = gateway.Generation
	gateway.Status.Replicas = 1
	gateway.Status.ReadyReplicas = 1
	Expect(k8sClient.Status().Update(ctx, gateway)).To(BeNil())

	hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
	Expect(updateStatus{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())
}
