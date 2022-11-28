package controllers

import (
	"context"
	"errors"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type mockReconcile struct {
	rq *requeue
}

func (m mockReconcile) reconcile(_ context.Context, _ *HStreamDBReconciler, _ *appsv1alpha1.HStreamDB) *requeue {
	return m.rq
}

var _ = Describe("HstreamdbController", func() {
	var mockRec *mockReconcile
	var hdb *appsv1alpha1.HStreamDB
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
		Expect(res).To(Equal(ctrl.Result{Requeue: true}))
	})

	It("should requeue delay", func() {
		mockRec.rq = &requeue{message: "requeue delay", delay: time.Second}
		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{Requeue: true, RequeueAfter: time.Second}))
	})

	It("should requeue delay with conflict error", func() {
		gr := schema.ParseGroupResource("app.hstream.io")
		conflictErr := k8sErrors.NewConflict(gr, "conflict", errors.New("something wrong"))
		mockRec.rq = &requeue{curError: conflictErr, delay: 0}

		res, err := clusterReconciler.subReconcile(ctx, hdb, subReconcilers)
		Expect(err).To(Succeed())
		Expect(res).To(Equal(ctrl.Result{Requeue: true, RequeueAfter: time.Minute}))
	})
})
