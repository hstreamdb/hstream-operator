package controllers

import (
	"context"
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct{}

func (u updateStatus) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "update status")

	if err := u.checkComponentsReady(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}

	if err := u.checkAllReady(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}

	logger.Info("Update status")
	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: err}
	}

	if !hdb.IsConditionTrue(hapi.Ready) {
		return &requeue{message: "HStreamDB is not ready", delayedRequeue: true}
	}

	return nil
}

func (u updateStatus) checkComponentsReady(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) error {
	componentsMap := map[hapi.ComponentType]string{
		hapi.ComponentTypeConsole: hapi.ConsoleReady,
		hapi.ComponentTypeGateway: hapi.GatewayReady,
	}

	for component, condition := range componentsMap {
		deploy := &appsv1.Deployment{}
		deploy.ObjectMeta = internal.GetObjectMetadata(hdb, nil, component)
		err := r.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)
		if err != nil {
			return err
		}
		hdb.SetCondition(metav1.Condition{
			Type:    condition,
			Status:  metav1.ConditionTrue,
			Reason:  condition,
			Message: fmt.Sprintf("%s is ready", component),
		})
	}
	return nil
}

func (u updateStatus) checkAllReady(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) error {
	var notReady = 0
	var condition = metav1.Condition{
		Type:   hapi.Ready,
		Status: metav1.ConditionFalse,
		Reason: "ComponentsNotReady",
	}

	conditionList := []string{
		hapi.HMetaReady,
		hapi.HStoreReady,
		hapi.HServerReady,
		hapi.GatewayReady,
		hapi.ConsoleReady,
	}
	for _, t := range conditionList {
		if !hdb.IsConditionTrue(t) {
			notReady++
		}
	}

	if notReady == 0 {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "AllComponentsReady"
		condition.Message = "All components are ready"
	}
	hdb.SetCondition(condition)
	return nil
}
