package controllers

import (
	"context"
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct{}

func (a updateStatus) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "update status")

	if err := a.updateConditions(ctx, r, hdb); err != nil {
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

func (a updateStatus) updateConditions(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) error {
	var notReady = 0
	var condition = metav1.Condition{
		Type:   hapi.Ready,
		Status: metav1.ConditionFalse,
		Reason: "ComponentsNotReady",
	}

	componentTypeList := []hapi.ComponentType{hapi.ComponentTypeHServer, hapi.ComponentTypeHStore}
	if hdb.Spec.Gateway != nil {
		componentTypeList = append(componentTypeList, hapi.ComponentTypeGateway)
	}

	for _, componentType := range componentTypeList {
		isReady, err := a.updateComponentCondition(ctx, r, hdb, componentType)
		if err != nil {
			return err
		}
		if !isReady {
			notReady++
			condition.Message = fmt.Sprintf("%s is not ready", componentType)
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

func (a updateStatus) updateComponentCondition(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB, componentType hapi.ComponentType) (isReady bool, err error) {
	var obj client.Object
	var bootstrapped bool
	var condition metav1.Condition = metav1.Condition{
		Status:  metav1.ConditionFalse,
		Reason:  "ComponentNotReady",
		Message: fmt.Sprintf("%s is not ready", componentType),
	}

	switch componentType {
	case hapi.ComponentTypeHStore:
		obj = &appsv1.StatefulSet{
			ObjectMeta: internal.GetObjectMetadata(hdb, nil, componentType),
		}
		condition.Type = hapi.HStoreReady
		bootstrapped = hdb.Status.HStore.Bootstrapped
	case hapi.ComponentTypeHServer:
		obj = &appsv1.StatefulSet{
			ObjectMeta: internal.GetObjectMetadata(hdb, nil, componentType),
		}
		condition.Type = hapi.HServerReady
		bootstrapped = hdb.Status.HServer.Bootstrapped
	case hapi.ComponentTypeGateway:
		obj = &appsv1.Deployment{
			ObjectMeta: internal.GetObjectMetadata(hdb, nil, componentType),
		}
		condition.Type = hapi.GatewayReady
		// Gateway is dependence on HServer
		bootstrapped = hdb.Status.HServer.Bootstrapped
	default:
		panic(fmt.Sprintf("unknown component type: %s", componentType))
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get component: %w", err)
	}
	if bootstrapped {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			if deployment.Status.ObservedGeneration == deployment.Generation && deployment.Status.ReadyReplicas == deployment.Status.Replicas {
				isReady = true
			}
		}
		if sts, ok := obj.(*appsv1.StatefulSet); ok {
			if sts.Status.ObservedGeneration == sts.Generation && sts.Status.ReadyReplicas == sts.Status.Replicas {
				isReady = true
			}
		}
	}
	if isReady {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "ComponentReady"
		condition.Message = fmt.Sprintf("%s is ready", componentType)
	}
	hdb.SetCondition(condition)
	return isReady, nil
}
