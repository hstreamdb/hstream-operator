package controllers

import (
	"context"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/admin"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type checkHMetaStatus struct{}

func (a checkHMetaStatus) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "check hmeta cluster")

	var err error

	if hdb.Spec.ExternalHMeta == nil {
		// determine if all hmeta pods are running
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: hapi.ComponentTypeHMeta.GetResName(hdb.Name),
			},
		}
		if err = checkPodRunningStatus(ctx, r.Client, hdb, sts); err != nil {
			// print message only to log, wait for reconciling after several second
			return &requeue{message: err.Error(), delay: time.Second}
		}
	}

	var cluster admin.HMetaStatus
	if cluster, err = r.AdminClientProvider.GetAdminClient(hdb).GetHMetaStatus(); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}
	if !cluster.IsAllReady() {
		return &requeue{message: "wait for hmeta cluster to be ready", delay: time.Second}
	}

	logger.Info("HMete is ready")
	hdb.Status.HMeta.Nodes = make([]hapi.HMetaNode, 0, len(cluster.Nodes))
	for id, node := range cluster.Nodes {
		hdb.Status.HMeta.Nodes = append(hdb.Status.HMeta.Nodes, hapi.HMetaNode{
			NodeId:    id,
			Reachable: node.Reachable,
			Leader:    node.Leader,
			Error:     node.Error,
		})
	}
	return nil
}
