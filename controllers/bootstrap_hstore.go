package controllers

import (
	"context"
	"errors"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type bootstrapHStore struct{}

func (a bootstrapHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "bootstrap hstore")

	if hdb.Status.HStoreConfigured {
		logger.Info("HStore has been bootstrapped before")
		return nil
	}

	// determine if all hstore pods are running
	comp := appsv1alpha1.ComponentTypeHStore
	if err := checkPodRunningStatus(ctx, r, hdb, *hdb.Spec.HStore.Replicas, comp); err != nil {
		// we only set the message to log, and reconcile after several second
		return &requeue{message: err.Error(), delay: 5 * time.Second}
	}

	// we bootstrap hstore through admin server, so we need to get the host of admin server service
	ip, port, err := a.getAdminServerHost(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	adminClient := r.AdminClientProvider.GetAdminClient(hdb)
	logger.Info("Check hstore status")
	if status, err := adminClient.GetStatus(ip, port); err != nil {
		return &requeue{curError: err, delay: 5 * time.Second}
	} else if status.HStoreInited {
		hdb.Status.HStoreConfigured = true
		return nil
	}

	logger.Info("Bootstrap hstore")
	// TODO we may check status before bootstrapping
	if err = adminClient.BootstrapHStore(ip, port); err != nil {
		return &requeue{curError: err, delay: 10 * time.Second}
	}

	hdb.Status.HStoreConfigured = true

	// we still need to delay several second before deploying hServer
	// while the first time we bootstrap successfully
	return &requeue{delay: 10 * time.Second}
}

func (a bootstrapHStore) getAdminServerHost(hdb *appsv1alpha1.HStreamDB) (ip string, port int, err error) {
	service := internal.GetService(hdb, nil, appsv1alpha1.ComponentTypeAdminServer)
	ports := mergePorts(adminServerPorts, hdb.Spec.AdminServer.Container.Ports)
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

func checkPodRunningStatus(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB,
	desiredReplicas int32, comp appsv1alpha1.ComponentType) error {

	pods, err := getPods(ctx, r, hdb, getPodListOptions(hdb, comp)...)
	if err != nil {
		return err
	}

	// TODO: compare replica with the desire number
	count := len(pods)
	if count == 0 || count < int(desiredReplicas) {
		return fmt.Errorf("%s cluster isn't ready", comp)
	}

	for i := range pods {
		pod := &pods[i]
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("%s cluster isn't ready", comp)
		}
	}
	return nil
}

func getPods(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB, options ...client.ListOption) (
	[]corev1.Pod, error) {

	pods := &corev1.PodList{}
	err := r.List(ctx, pods, options...)
	if err != nil {
		return nil, err
	}

	// TODO: change hdb to the sts/deployment that related to the component
	// determine if a resource is related to this cluster.
	/*resPods := make([]corev1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		for _, reference := range pod.ObjectMeta.OwnerReferences {
			if reference.UID == hdb.UID {
				resPod := pod
				resPods = append(resPods, resPod)
				break
			}
		}
	}
	return resPods, nil*/
	return pods.Items, nil
}

func getPodListOptions(hdb *appsv1alpha1.HStreamDB, component appsv1alpha1.ComponentType) []client.ListOption {
	return []client.ListOption{
		client.InNamespace(hdb.Namespace),
		client.MatchingLabels{
			appsv1alpha1.ComponentKey: string(component),
		},
	}
}
