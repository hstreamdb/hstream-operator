package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/hstreamdb/hstream-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RqliteStatusReconciler struct{}

func (RqliteStatusReconciler) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB) *requeue {
	var err error

	internalRqliteStatus, err := r.AdminClientProvider.GetAdminClient(hdb).GetInternalRqliteStatus()
	if err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}
	if !internalRqliteStatus.IsAllNodesReady() {
		return &requeue{message: "wait for rqlite to be ready", delay: time.Second}
	}

	hdb.Status.InternalRqlite.Nodes = internalRqliteStatus.Nodes

	hdb.SetCondition(metav1.Condition{
		Type:    v1alpha2.InternalRqliteReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha2.InternalRqliteReady,
		Message: "Rqlite is ready",
	})

	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("failed to update hstreamdb status: %w", err)}
	}

	return nil
}
