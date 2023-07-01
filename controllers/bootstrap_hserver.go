package controllers

import (
	"context"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHServer struct{}

func (a bootstrapHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hServer")

	if hdb.Status.HServer.Bootstrapped {
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

	hdb.Status.HServer.Bootstrapped = true
	return nil
}
