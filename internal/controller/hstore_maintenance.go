package controller

import (
	"context"
	"strconv"
	"strings"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/admin"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HStoreMaintenanceReconciler struct{}

func (r HStoreMaintenanceReconciler) reconcile(ctx context.Context, hr *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "HStoreMaintenanceReconciler")

	if !hdb.IsConditionTrue(hapi.HStoreUpdating) {
		return nil
	}

	existingSts := appsv1.StatefulSet{}
	err := hr.Client.Get(ctx, types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHStore.GetResName(hdb.Name),
	}, &existingSts)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return &requeue{curError: err}
	}

	oldReplicas, _ := strconv.ParseInt(existingSts.Annotations["oldReplicas"], 10, 32)

	exist, condition := hdb.GetCondition(hapi.HStoreUpdating)
	if exist > -1 && condition.Reason == hapi.HStoreScalingDown {
		logger.Info("Start to drain HStore")

		if err = hr.AdminClientProvider.GetHAdminClient(hdb).MaintenanceHStore(
			admin.MaintenanceActionApply,
			[]string{
				"--node-indexes", r.getDrainedNodeIndexes(hdb, int(oldReplicas)),
				"--shard-target-state", "drained"}); err != nil {
			return &requeue{curError: err}
		}
	}

	return nil
}

func (r *HStoreMaintenanceReconciler) getDrainedNodeIndexes(hdb *hapi.HStreamDB, oldReplicas int) string {
	replicas := hdb.Spec.HStore.Replicas
	var indexes []string

	for i := 0; i < oldReplicas; i++ {
		indexes = append(indexes, strconv.Itoa(i))
	}

	return strings.Join(indexes[replicas:], " ")
}
