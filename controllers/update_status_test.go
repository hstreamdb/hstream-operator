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

	Context("with a new cluster", func() {
		nodes := []hapi.HMetaNode{
			{
				NodeId:    "node-id",
				Reachable: true,
				Leader:    false,
				Error:     "",
			},
		}
		JustBeforeEach(func() {
			hdb.Status.HMeta.Nodes = nodes
			hdb.Status.HStore.Bootstrapped = true
			hdb.Status.HServer.Bootstrapped = true
			_ = updateStatus.reconcile(ctx, clusterReconciler, hdb)
		})

		It("check store resource", func() {
			storeHdb := &hapi.HStreamDB{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
			Expect(storeHdb.Status.HMeta.Nodes).To(ConsistOf(nodes))
			Expect(storeHdb.Status.HStore.Bootstrapped).To(BeTrue())
			Expect(storeHdb.Status.HServer.Bootstrapped).To(BeTrue())
		})

	})

	Context("check conditions", func() {
		When("just hStore has been ready", Label("conditions"), func() {
			JustBeforeEach(func() {
				prepareHStoreReady(ctx, hdb)
				hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
				hdb.Status.HStore.Bootstrapped = true
				hdb.Status.HServer.Bootstrapped = false
			})

			It("check conditions", func() {
				Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).NotTo(BeNil())
				storeHdb := &hapi.HStreamDB{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
				Expect(storeHdb.Status.Conditions).To(ContainElements(
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
				hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
				hdb.Status.HStore.Bootstrapped = false
				hdb.Status.HServer.Bootstrapped = true
			})

			It("check conditions", func() {
				Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).NotTo(BeNil())
				storeHdb := &hapi.HStreamDB{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
				Expect(storeHdb.Status.Conditions).To(ContainElements(
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
				hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
				hdb.Status.HStore.Bootstrapped = false
				hdb.Status.HServer.Bootstrapped = true
			})

			It("check conditions", func() {
				Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).NotTo(BeNil())
				storeHdb := &hapi.HStreamDB{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
				Expect(storeHdb.Status.Conditions).To(ContainElements(
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
				hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
				hdb.Status.HStore.Bootstrapped = true
				hdb.Status.HServer.Bootstrapped = true
			})

			It("check conditions", func() {
				Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())
				storeHdb := &hapi.HStreamDB{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
				Expect(storeHdb.Status.Conditions).To(ConsistOf(
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
}

func prepareGatewayReady(ctx context.Context, hdb *hapi.HStreamDB) {
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
}
