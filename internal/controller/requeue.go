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
	// delay provides an optional delay before requeueing reconciliation.
	delay time.Duration

	// curError provides an error that we encountered that forced a requeue.
	curError error

	// message provides a log message that explains the reason for the requeue.
	message string

	// delayedRequeue defines that the reconciliation was not completed but the requeue should be delayed to the end.
	delayedRequeue bool
}

// processRequeue interprets a requeue result from a subreconciler.
func processRequeue(requeue *requeue, subReconciler interface{}, object runtime.Object,
	recorder record.EventRecorder, logger logr.Logger) (ctrl.Result, error) {

	curLog := logger.WithValues("subReconciler",
		fmt.Sprintf("%T", subReconciler), "requeueAfter", requeue.delay)

	if requeue.message == "" && requeue.curError != nil {
		requeue.message = requeue.curError.Error()
	}

	err := requeue.curError
	if err != nil && k8sErrors.IsConflict(err) {
		err = nil
		if requeue.delay == time.Duration(0) {
			requeue.delay = time.Second
		}
	}

	recorder.Event(object, corev1.EventTypeNormal, "ReconciliationTerminatedEarly", requeue.message)

	if err != nil {
		curLog.Error(err, "Error in reconciliation")
		return ctrl.Result{}, err
	}
	curLog.V(1).Info("Reconciliation terminated early", "message", requeue.message)

	return ctrl.Result{RequeueAfter: requeue.delay}, nil
}
