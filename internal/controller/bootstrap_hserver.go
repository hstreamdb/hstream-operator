package controller

import (
	"context"
	"fmt"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/status"
	"github.com/hstreamdb/hstream-operator/internal/utils"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type bootstrapHServer struct{}

func (a bootstrapHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hServer")

	if hdb.IsConditionTrue(hapi.HServerReady) {
		return nil
	}

	// TODO: change sts to deployment
	// determine if all hServer pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: hapi.ComponentTypeHServer.GetResName(hdb),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &hdb.Spec.HServer.Replicas,
		},
	}
	if err := status.CheckReplicasReadyStatus(ctx, r.Client, hdb, sts); err != nil {
		// we only set the message to log, and reconcile after several second
		return &requeue{message: err.Error(), delay: time.Second}
	}

	logger.Info("Bootstrap hServer")

	client := r.AdminClientProvider.GetAdminClient(hdb)
	if hdb.Spec.Config.KafkaMode {
		port := utils.FindContainerPortByName(sts.Spec.Template.Spec.Containers[0].Ports, constants.DefaultHServerPort.Name)

		if _, err := client.CallKafkaServer(
			"init",
			"--host", hapi.ComponentTypeHServer.GetInternalResName(hdb),
			"--port", fmt.Sprintf("%d", port.ContainerPort),
		); err != nil {
			return &requeue{message: err.Error(), delay: time.Second * 5}
		}
	} else {
		if _, err := client.CallServer(
			"init",
			"--host", hapi.ComponentTypeHServer.GetInternalResName(hdb),
		); err != nil {
			return &requeue{message: err.Error(), delay: time.Second * 5}
		}
	}

	hdb.SetCondition(metav1.Condition{
		Type:    hapi.HServerReady,
		Status:  metav1.ConditionTrue,
		Reason:  hapi.HServerReady,
		Message: "HServer has been bootstrapped",
	})
	logger.Info("Update HServer status")
	if err := r.Status().Update(ctx, hdb); err != nil {
		return &requeue{curError: fmt.Errorf("update HStore status failed: %w", err)}
	}
	return nil
}
