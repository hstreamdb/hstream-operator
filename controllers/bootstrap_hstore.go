package controllers

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	jsoniter "github.com/json-iterator/go"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bootstrapHStore struct{}

func (a bootstrapHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hstore")

	if hdb.Status.HStore.Bootstrapped {
		logger.Info("HStore has been bootstrapped before")
		return nil
	}

	var err error

	// determine if all hstore pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHStore.GetResName(hdb.Name),
		},
	}
	if err = checkPodRunningStatus(ctx, r.Client, hdb, sts); err != nil {
		// print message only to log, wait for reconciling after several second
		return &requeue{message: err.Error(), delay: 10 * time.Second}
	}

	logger.Info("Bootstrap hstore")

	metadataReplication := int32(0)
	if hdb.Spec.Config.MetadataReplicateAcross == nil || *hdb.Spec.Config.MetadataReplicateAcross > hdb.Spec.HStore.Replicas {
		metadataReplication = getRecommendedLogReplicaAcross(hdb.Spec.HStore.Replicas)
	} else {
		metadataReplication = *hdb.Spec.Config.MetadataReplicateAcross
	}
	if err = r.AdminClientProvider.GetAdminClient(hdb).BootstrapHStore(metadataReplication); err != nil {
		return &requeue{message: err.Error(), delay: 10 * time.Second}
	}

	hdb.Status.HStore.Bootstrapped = true
	logger.Info("Update hstore status")
	if err = r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}

	// we still need to delay several second before deploying hServer
	// while the first time we bootstrap successfully
	return &requeue{delay: 10 * time.Second}
}

func checkPodRunningStatus(ctx context.Context, client client.Client, hdb *hapi.HStreamDB, obj client.Object) error {
	count, err := getReadyReplicasInService(ctx, client, hdb, obj)
	if err != nil {
		return err
	}

	var desiredReplicas int32
	if sts, ok := obj.(*appsv1.StatefulSet); ok {
		desiredReplicas = *sts.Spec.Replicas
	} else if deploy, ok := obj.(*appsv1.Deployment); ok {
		desiredReplicas = *deploy.Spec.Replicas
	} else {
		err = fmt.Errorf("%s is neither StatefulSet nor Deployment", obj.GetName())
		return err
	}

	if count == 0 || count < int(desiredReplicas) {
		return fmt.Errorf("%s cluster isn't ready", obj.GetName())
	}
	return nil
}

func getReadyReplicasInService(ctx context.Context, client client.Client, hdb *hapi.HStreamDB,
	obj client.Object) (count int, err error) {

	err = client.Get(ctx, types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      obj.GetName(),
	}, obj)
	if err != nil {
		return
	}

	bin, _ := jsoniter.Marshal(obj)
	ret := jsoniter.Get(bin, "status", "readyReplicas")
	if ret.LastError() != nil {
		err = fmt.Errorf("wait for all of %s nodes to be ready", obj.GetName())
		return
	}

	count = ret.ToInt()
	return
}
