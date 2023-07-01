/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/admin"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("HStreamDB Controller")

// HStreamDBReconciler reconciles a HStreamDB object
type HStreamDBReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Recorder            record.EventRecorder
	AdminClientProvider admin.AdminClientProvider
}

type hdbSubReconciler interface {
	/**
	reconcile runs the reconciler's work.

	If reconciliation can continue, this should return nil.

	If reconciliation encounters an error, this should return a	requeue object
	with an `Error` field.

	If reconciliation cannot proceed, this should return a requeue object with
	a `Message` field.
	*/
	reconcile(ctx context.Context, r *HStreamDBReconciler, cluster *hapi.HStreamDB) *requeue
}

//+kubebuilder:rbac:groups=apps.hstream.io,resources=hstreamdbs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.hstream.io,resources=hstreamdbs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.hstream.io,resources=hstreamdbs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HStreamDB object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *HStreamDBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	//_ = log.FromContext(ctx)

	hdb := &hapi.HStreamDB{}
	if err = r.Get(ctx, req.NamespacedName, hdb); err != nil {
		if k8sErrors.IsNotFound(err) {
			err = nil
		}
		// Error reading the object - requeue the request.
		return
	}

	subReconcilers := []hdbSubReconciler{
		updateConfigMap{},
		addServices{},
		addHMeta{},
		checkHMetaStatus{},
		addAdminServer{},
		addHStore{},
		bootstrapHStore{},
		addHServer{},
		bootstrapHServer{},
		addGateway{},
		addConsole{},
		updateStatus{},
	}
	return r.subReconcile(ctx, hdb, subReconcilers)
}

func (r *HStreamDBReconciler) subReconcile(ctx context.Context, hdb *hapi.HStreamDB, subReconcilers []hdbSubReconciler) (
	ctrl.Result, error) {

	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name)

	delayedRequeue := false
	for _, subReconciler := range subReconcilers {
		logger.V(1).Info("Attempting to run sub-reconciler", "subReconciler", fmt.Sprintf("%T", subReconciler))
		requeue := subReconciler.reconcile(ctx, r, hdb)
		if requeue == nil {
			continue
		}

		if requeue.delayedRequeue {
			logger.V(1).Info("Delaying requeue for sub-reconciler",
				"subReconciler", fmt.Sprintf("%T", subReconciler),
				"message", requeue.message,
				"error", requeue.curError)
			delayedRequeue = true
			continue
		}
		return processRequeue(requeue, subReconciler, hdb, r.Recorder, logger)
	}

	if delayedRequeue {
		logger.V(1).Info("HStream was not fully reconciled by reconciliation process")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	logger.Info("Reconciliation complete")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "ReconciliationComplete", "")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HStreamDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hapi.HStreamDB{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to CR status in which case metadata.Generation does not change
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}
