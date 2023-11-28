package controller

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/controller/status"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHStore struct{}

func (a bootstrapHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap HStore")

	if hdb.IsConditionTrue(hapi.HStoreReady) || hdb.IsConditionTrue(hapi.HStoreUpdating) {
		return nil
	}

	var err error

	// Determine if all HStore pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHStore.GetResName(hdb.Name),
		},
	}
	if err = status.CheckReplicasReadyStatus(ctx, r.Client, hdb, sts); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}

	logger.Info("Bootstraping HStore")

	metadataReplication := int32(0)
	if hdb.Spec.Config.MetadataReplicateAcross == nil || *hdb.Spec.Config.MetadataReplicateAcross > hdb.Spec.HStore.Replicas {
		metadataReplication = getRecommendedLogReplicaAcross(hdb.Spec.HStore.Replicas)
	} else {
		metadataReplication = *hdb.Spec.Config.MetadataReplicateAcross
	}

	if _, err = r.AdminClientProvider.GetHAdminClient(hdb).CallStore(
		"nodes-config", "bootstrap",
		"--metadata-replicate-across", fmt.Sprintf("node:%d", metadataReplication),
	); err != nil {
		return &requeue{delay: time.Second}
	}

	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HStoreReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HStoreReady,
		Message: "All HStore nodes have been bootstrapped",
	})

	if err = r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}

	// We still need to delay several seconds to deploy HServer component
	// after the first time we bootstrap successfully.
	return &requeue{delay: time.Second}
}
