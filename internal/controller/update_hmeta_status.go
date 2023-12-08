package controller

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/admin"
	"github.com/hstreamdb/hstream-operator/internal/controller/status"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type updateHMetaStatus struct{}

func (u updateHMetaStatus) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	var err error

	if hdb.Spec.ExternalHMeta == nil {
		// Determine if all HMeta pods are running.
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: hapi.ComponentTypeHMeta.GetResName(hdb.Name),
			},
		}
		if err = status.CheckReplicasReadyStatus(ctx, r.Client, hdb, sts); err != nil {
			return &requeue{message: err.Error(), delay: time.Second}
		}
	}

	var cluster admin.HMetaStatus
	if cluster, err = r.AdminClientProvider.GetAdminClient(hdb).GetHMetaStatus(); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}
	if !cluster.IsAllReady() {
		return &requeue{message: "wait for HMeta cluster to be ready", delay: time.Second}
	}

	hdb.Status.HMeta.Nodes = make([]hapi.HMetaNode, 0, len(cluster.Nodes))
	for id, node := range cluster.Nodes {
		hdb.Status.HMeta.Nodes = append(hdb.Status.HMeta.Nodes, hapi.HMetaNode{
			NodeId:    id,
			Reachable: node.Reachable,
			Leader:    node.Leader,
			Error:     node.Error,
		})
	}
	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HMetaReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HMetaReady,
		Message: "HMeta is ready",
	})
	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HMeta status failed: %w", err)}
	}
	return nil
}
