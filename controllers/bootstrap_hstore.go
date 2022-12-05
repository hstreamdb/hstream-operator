package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	jsoniter "github.com/json-iterator/go"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type bootstrapHStore struct{}

func (a bootstrapHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hstore")

	if hdb.Status.HStoreConfigured {
		logger.Info("HStore has been bootstrapped before")
		return nil
	}

	var err error

	// determine if all hstore pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: appsv1alpha1.ComponentTypeHStore.GetResName(hdb.Name),
		},
	}
	if err = checkPodRunningStatus(ctx, r.Client, hdb, sts); err != nil {
		// we only set the message to log, and reconcile after several second
		return &requeue{message: err.Error(), delay: 10 * time.Second}
	}

	logger.Info("Bootstrap hstore")
	if err = r.AdminClientProvider.GetAdminClient(hdb).BootstrapHStore(); err != nil {
		return &requeue{message: err.Error(), delay: 10 * time.Second}
	}

	hdb.Status.HStoreConfigured = true
	logger.Info("Update status")
	if err = r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}

	// we still need to delay several second before deploying hServer
	// while the first time we bootstrap successfully
	return &requeue{delay: 10 * time.Second}
}

func checkPodRunningStatus(ctx context.Context, client client.Client, hdb *appsv1alpha1.HStreamDB, obj client.Object) error {
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

func getReadyReplicasInService(ctx context.Context, client client.Client, hdb *appsv1alpha1.HStreamDB,
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
		err = fmt.Errorf("couldn't get ready replicas from status of %s: %w", obj.GetName(), ret.LastError())
		return
	}

	count = ret.ToInt()
	return
}
