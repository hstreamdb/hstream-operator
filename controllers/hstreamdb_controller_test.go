package controllers

import (
	"context"
	"errors"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// this test case requires to connect to the running k8s cluster in local or anywhere
// set env: export USE_EXISTING_CLUSTER=true
// run the test: ginkgo run --label-filter 'k8s' controllers/
var _ = Describe("BootstrapHServer", Label("k8s"), func() {
	timeout := 3 * time.Minute

	var hdb *hapi.HStreamDB
	ctx := context.TODO()

	BeforeEach(func() {
		if !isUsingExistingCluster() {
			Skip("Skip testcase BootstrapHServer")
		}

		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
		Eventually(func() bool {
			existHDB := &hapi.HStreamDB{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), existHDB)
			return k8sErrors.IsNotFound(err)
		}, timeout, 10*time.Second).Should(BeTrue())
	})

	It("check status", func() {
		Eventually(func() bool {
			existHDB := &hapi.HStreamDB{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), existHDB)
			if err == nil {
				return existHDB.Status.HServer.Bootstrapped == true &&
					existHDB.Status.HStore.Bootstrapped == true
			}
			return false
		}, timeout, 10*time.Second).Should(BeTrue())
	})
})

var _ = Describe("HstreamdbController", func() {
	var mockRec *mockReconcile
	var hdb *hapi.HStreamDB
	var subReconcilers []hdbSubReconciler
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		mockRec = &mockReconcile{}
		subReconcilers = []hdbSubReconciler{
			mockRec,
		}
	})

	It("no requeue", func() {
		mockRec.rq = nil
		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{}))
	})

	It("reconcile failed", func() {
		mockRec.rq = &requeue{curError: errors.New("mock requeue")}
		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(HaveOccurred())
		Expect(res).To(Equal(ctrl.Result{}))
	})

	It("should requeue next time immediately", func() {
		mockRec.rq = &requeue{delayedRequeue: true}
		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: time.Second}))
	})

	It("should requeue delay", func() {
		mockRec.rq = &requeue{message: "requeue delay", delay: time.Second}
		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: time.Second}))
	})

	It("should requeue delay with conflict error", func() {
		gr := schema.ParseGroupResource("app.hstream.io")
		conflictErr := k8sErrors.NewConflict(gr, "conflict", errors.New("something wrong"))
		mockRec.rq = &requeue{curError: conflictErr, delay: 0}

		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: time.Second}))
	})
})

type mockReconcile struct {
	rq *requeue
}

func (m mockReconcile) reconcile(_ context.Context, _ *HStreamDBReconciler, _ *hapi.HStreamDB) *requeue {
	return m.rq
}
