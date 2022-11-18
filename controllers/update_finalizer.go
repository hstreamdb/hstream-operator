package controllers

import (
	"context"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizer = "apps.hstream.io/finalizer"
)

type updateFinalizer struct{}

func (u updateFinalizer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	hasRegFinalizer := controllerutil.ContainsFinalizer(hdb, finalizer)

	// examine DeletionTimestamp to determine if object is under deletion
	if !hdb.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if hasRegFinalizer {
			// TODO: do something?
			controllerutil.RemoveFinalizer(hdb, finalizer)
			if err := r.Client.Update(ctx, hdb); err != nil {
				return &requeue{curError: err}
			}
		}
		// Stop reconciliation as the item is being deleted
		return &requeue{}
	}

	// The object is not being deleted
	if !hasRegFinalizer {
		controllerutil.AddFinalizer(hdb, finalizer)
		if err := r.Update(ctx, hdb); err != nil {
			return &requeue{curError: err}
		}
	}
	return nil
}
