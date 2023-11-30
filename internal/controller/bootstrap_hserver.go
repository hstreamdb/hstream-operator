package controller

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/internal/controller/status"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHServer struct{}

func (a bootstrapHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap HServer")

	if hdb.IsConditionTrue(hapi.HServerReady) {
		return nil
	}

	// TODO: change sts to deployment.
	// Determine if all HServer pods are running.
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHServer.GetResName(hdb.Name),
		},
	}

	if err := status.CheckReplicasReadyStatus(ctx, r.Client, hdb, sts); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}

	logger.Info("Bootstrap HServer")

	if _, err := r.AdminClientProvider.GetHAdminClient(hdb).CallServer(
		"server", "init",
		"--host", internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer).Name,
	); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}

	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HServerReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HServerReady,
		Message: "All HServer nodes have been bootstrapped",
	})

	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}

	return nil
}
