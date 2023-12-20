package controller

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// requeue provides a wrapper around different results from a subreconciler.
type requeue struct {
	// delay defines an optional delay before requeueing reconciliation.
	delay time.Duration

	// curError defines an error that we encountered that forced a requeue.
	curError error

	// message defines a log message that explains the reason for the requeue.
	message string

	// delayedRequeue defines that the reconciliation was not completed but the requeue should be delayed to the end.
	delayedRequeue bool
}

// processRequeue interprets a requeue result from a subreconciler.
func processRequeue(requeue *requeue, subReconciler interface{}, object runtime.Object,
	recorder record.EventRecorder, logger logr.Logger) (ctrl.Result, error) {
	log := logger.WithValues("sub-reconciler", fmt.Sprintf("%T", subReconciler), "delay", requeue.delay)

	err := requeue.curError
	if err != nil {
		if k8sErrors.IsConflict(err) {
			log.V(1).Info("conflict in reconciliation", "message", err.Error())

			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		log.Error(err, "error in reconciliation")

		return ctrl.Result{}, err
	}

	if requeue.message != "" && requeue.delay > 0 {
		recorder.Event(object, corev1.EventTypeNormal, "ReconciliationTerminatedEarly", requeue.message)
		log.V(1).Info("reconciliation terminated early", "message", requeue.message)
	}

	return ctrl.Result{RequeueAfter: requeue.delay}, nil
}
