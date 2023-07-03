package controllers

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHServer struct{}

func (a bootstrapHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hServer")

	if hdb.IsConditionTrue(hapi.HServerReady) {
		return nil
	}

	// TODO: change sts to deployment
	// determine if all hServer pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHServer.GetResName(hdb.Name),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &hdb.Spec.HServer.Replicas,
		},
	}
	if err := checkPodRunningStatus(ctx, r.Client, hdb, sts); err != nil {
		// we only set the message to log, and reconcile after several second
		return &requeue{message: err.Error(), delay: time.Second}
	}

	logger.Info("Bootstrap hServer")
	if err := r.AdminClientProvider.GetAdminClient(hdb).BootstrapHServer(); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}

	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HServerReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HServerReady,
		Message: "HServer has been bootstrapped",
	})
	logger.Info("Update HServer status")
	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}
	return nil
}
