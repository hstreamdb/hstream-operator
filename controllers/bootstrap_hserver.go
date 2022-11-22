package controllers

import (
	"context"
	"errors"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type bootstrapHServer struct{}

func (a bootstrapHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hServer")

	if hdb.Status.HServerConfigured {
		logger.Info("HServer has been bootstrapped before")
		return nil
	}

	// TODO: change sts to deployment
	// determine if all hServer pods are running
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: appsv1alpha1.ComponentTypeHServer.GetResName(hdb.Name),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: hdb.Spec.HServer.Replicas,
		},
	}
	if err := checkPodRunningStatus(ctx, r, hdb, sts); err != nil {
		// we only set the message to log, and reconcile after several second
		return &requeue{message: err.Error(), delay: 5 * time.Second}
	}

	ip, port, err := a.getHServerHost(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	logger.Info("Bootstrap hServer")
	if err = r.AdminClientProvider.GetAdminClient(hdb).BootstrapHServer(ip, port); err != nil {
		return &requeue{curError: err, delay: 10 * time.Second}
	}

	hdb.Status.HServerConfigured = true
	return nil
}

func (a bootstrapHServer) getHServerHost(hdb *appsv1alpha1.HStreamDB) (ip string, port int, err error) {
	service := internal.GetHeadlessService(hdb, appsv1alpha1.ComponentTypeHServer)

	ports := mergePorts(hServerPorts, hdb.Spec.HServer.Container.Ports)
	for _, p := range ports {
		if p.Name == "port" {
			port = int(p.ContainerPort)
		}
	}

	if port == 0 {
		err = errors.New("invalid port")
		return
	}
	return service.Name, port, nil
}
