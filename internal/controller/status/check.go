package status

import (
	"context"
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckReplicasReadyStatus(ctx context.Context, client client.Client, hdb *hapi.HStreamDB, obj client.Object) error {
	err := client.Get(ctx, types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      obj.GetName(),
	}, obj)
	if err != nil {
		return err
	}

	switch obj.(type) {
	case *appsv1.StatefulSet:
	case *appsv1.Deployment:
		desiredReplicas := obj.(*appsv1.StatefulSet).Spec.Replicas
		statusReplicas := obj.(*appsv1.StatefulSet).Status.Replicas
		statusReadyReplicas := obj.(*appsv1.StatefulSet).Status.ReadyReplicas

		if *desiredReplicas == statusReplicas && *desiredReplicas == statusReadyReplicas {
			return nil
		} else {
			return fmt.Errorf("ready replicas is not equal to desired replicas in %s", obj.GetName())
		}
	default:
		return fmt.Errorf("%s is neither StatefulSet nor Deployment", obj.GetName())
	}

	return nil
}
