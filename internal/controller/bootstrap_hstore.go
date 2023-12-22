package controller

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/status"
	"github.com/hstreamdb/hstream-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHStore struct{}

func (a bootstrapHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap HStore")

	if hdb.IsConditionTrue(hapi.HStoreReady) {
		return nil
	}

	var err error

	// determine if all HStore pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHStore.GetResName(hdb),
		},
	}
	if err := status.CheckReplicasReadyStatus(ctx, r.Client, hdb, sts); err != nil {
		return &requeue{message: err.Error(), delay: time.Second}
	}

	logger.Info("Bootstrap HStore")

	var metadataReplication int
	if hdb.Spec.Config.MetadataReplicateAcross == nil || *hdb.Spec.Config.MetadataReplicateAcross > hdb.Spec.HStore.Replicas {
		metadataReplication = utils.GetRecommendedLogReplicaAcross(hdb)
	} else {
		metadataReplication = int(*hdb.Spec.Config.MetadataReplicateAcross)
	}

	if _, err = r.AdminClientProvider.GetAdminClient(hdb).CallStore(
		"nodes-config", "bootstrap",
		"--metadata-replicate-across", fmt.Sprintf("node:%d", metadataReplication),
	); err != nil {
		return &requeue{message: err.Error(), delay: time.Second * 5}
	}

	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HStoreReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HStoreReady,
		Message: "HStore has been bootstrapped",
	})
	logger.Info("Update HStore status")
	if err = r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}

	// we still need to delay several second before deploying hServer
	// while the first time we bootstrap successfully
	return &requeue{delay: time.Second}
}
