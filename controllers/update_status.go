package controllers

import (
	"context"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct{}

func (a updateStatus) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "update status")

	var err error
	oldHdb := &appsv1alpha1.HStreamDB{}
	if err = r.Get(ctx, client.ObjectKeyFromObject(hdb), oldHdb); err != nil {
		if k8sErrors.IsNotFound(err) {
			err = nil
		}
		// Error reading the object - requeue the request.
		return &requeue{curError: err}
	}

	// TODO: this update operation will trigger k8s to reconcile again
	if !equality.Semantic.DeepEqual(oldHdb.Status, hdb.Status) {
		logger.Info("Update status")
		if err = r.Status().Update(ctx, hdb); err != nil {
			return &requeue{curError: err}
		}
	}
	return nil
}
