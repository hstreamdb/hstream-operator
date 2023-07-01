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

		prepareGatewayReady(ctx, hdb)
		prepareConsoleReady(ctx, hdb)
		hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
		// Delete all components to prevent contamination of other use cases
		_ = k8sClient.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace(hdb.Namespace))
		_ = k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(hdb.Namespace))
	})

	It("check conditions", func() {
		Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).To(Equal(&requeue{message: "HStreamDB is not ready", delayedRequeue: true}))
		storeHdb := &hapi.HStreamDB{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
		Expect(storeHdb.Status.Conditions).To(ContainElement(
			And(
				HaveField("Type", Equal(hapi.Ready)),
				HaveField("Status", Equal(metav1.ConditionFalse)),
			),
		))
	})

	When("all components have been ready", func() {
		JustBeforeEach(func() {
			hdb.Status.HMeta.Nodes = []hapi.HMetaNode{}
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.HMetaReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.HStoreReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.HServerReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.GatewayReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.ConsoleReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
		})

		It("check conditions", func() {
			Expect(updateStatus.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())
			storeHdb := &hapi.HStreamDB{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), storeHdb)).To(BeNil())
			Expect(storeHdb.Status.Conditions).To(ContainElement(
				And(
					HaveField("Type", Equal(hapi.Ready)),
					HaveField("Status", Equal(metav1.ConditionTrue)),
				),
			))
		})
	})
})

func prepareGatewayReady(ctx context.Context, hdb *hapi.HStreamDB) {
	hdb.Spec.Gateway = &hapi.Gateway{}
	hdb.Spec.Gateway.Endpoint = "fake"
	hdb.Spec.Gateway.Port = 14789
	hdb.Spec.Gateway.Image = "hstreamdb/hstream-gateway:latest"
	hdb.Spec.Gateway.Replicas = 1
	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HServerReady,
		Status:  metav1.ConditionTrue,
		Reason:  "test",
		Message: "test",
	})
	Expect(addGateway{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())

	gateway := &appsv1.Deployment{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeGateway),
	}
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(gateway), gateway)).To(BeNil())
	gateway.Status.Replicas = 1
	gateway.Status.ReadyReplicas = 1
	Expect(k8sClient.Status().Update(ctx, gateway)).To(BeNil())
}

func prepareConsoleReady(ctx context.Context, hdb *hapi.HStreamDB) {
	Expect(addConsole{}.reconcile(ctx, clusterReconciler, hdb)).To(BeNil())

	console := &appsv1.Deployment{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeConsole),
	}
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(console), console)).To(BeNil())
	console.Status.Replicas = 1
	console.Status.ReadyReplicas = 1
	Expect(k8sClient.Status().Update(ctx, console)).To(BeNil())
}
