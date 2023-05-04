package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addGateway struct{}

func (a addGateway) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	if hdb.Spec.Gateway == nil {
		return nil
	}

	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add gateway")
	if !hdb.Status.HServer.Bootstrapped {
		return &requeue{message: "wait for HServer nodes to be ready", delay: 10 * time.Second}
	}

	deploy := a.getDeployment(ctx, r, hdb)

	existingDeploy := &appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(&deploy), existingDeploy)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}
		if err = ctrl.SetControllerReference(hdb, &deploy, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		logger.Info("Create gateway")
		if err = r.Client.Create(ctx, &deploy); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}
	if !isHashChanged(&existingDeploy.ObjectMeta, &deploy.ObjectMeta) {
		return nil
	}

	logger.Info("Update gateway")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingGateway", "")

	existingDeploy.Annotations = deploy.Annotations
	existingDeploy.Labels = deploy.Labels
	existingDeploy.Spec = deploy.Spec
	if err = r.Update(ctx, existingDeploy); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addGateway) getDeployment(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) appsv1.Deployment {
	podTemplate := a.getPodTemplate(ctx, r, hdb)
	deploy := internal.GetDeployment(hdb, &hdb.Spec.Gateway.Component,
		&podTemplate, hapi.ComponentTypeGateway)

	return deploy
}

func (a addGateway) getPodTemplate(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) corev1.PodTemplateSpec {
	gateway := hdb.Spec.Gateway

	pod := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeGateway),
		Spec: corev1.PodSpec{
			Affinity:        gateway.Affinity,
			Tolerations:     gateway.Tolerations,
			NodeName:        gateway.NodeName,
			NodeSelector:    gateway.NodeSelector,
			SecurityContext: gateway.PodSecurityContext,
			InitContainers:  gateway.InitContainers,
			Containers:      a.getContainer(ctx, r, hdb),
			Volumes:         gateway.Volumes,
		},
	}

	pod.Name = hapi.ComponentTypeAdminServer.GetResName(hdb.Name)
	pod.Spec.Volumes = append(pod.Spec.Volumes, a.getVolumes(hdb)...)
	return pod
}

func (a addGateway) getContainer(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) []corev1.Container {
	gateway := hdb.Spec.Gateway
	container := corev1.Container{
		Image:           hdb.Spec.Gateway.Image,
		ImagePullPolicy: hdb.Spec.Gateway.ImagePullPolicy,
	}

	structAssign(&container, gateway.Container.DeepCopy())

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeGateway)
	}

	container.Ports = extendPorts(map[string]string{},
		container.Ports,
		[]corev1.ContainerPort{
			{Name: "port", ContainerPort: gateway.Port},
		},
	)

	podList := corev1.PodList{}
	_ = r.List(ctx, &podList,
		client.InNamespace(hdb.Namespace),
		client.MatchingLabels(
			internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHServer).Labels,
		),
	)

	port := findHServerPort(hdb, podList.Items[0])
	addresses := []string{}
	for _, pod := range podList.Items {
		addresses = append(addresses, fmt.Sprintf("hstream://%s:%d", pod.Status.PodIP, port))
	}

	extendEnv(&container, []corev1.EnvVar{
		{Name: "ENABLE_TLS", Value: "false"},
		{Name: "ENDPOINT_HOST", Value: gateway.Endpoint},
		{Name: "HSTREAM_SERVICE_URL", Value: strings.Join(addresses, ",")},
	})

	return append([]corev1.Container{container}, gateway.SidecarContainers...)
}

func (a addGateway) getVolumes(hdb *hapi.HStreamDB) (volumes []corev1.Volume) {
	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	volumes = []corev1.Volume{internal.GetVolume(hdb, m)}
	return
}

func findHServerPort(hdb *hapi.HStreamDB, pod corev1.Pod) int32 {
	hServerContainerName := hdb.Spec.HServer.Container.Name
	if hServerContainerName == "" {
		hServerContainerName = string(hapi.ComponentTypeHServer)
	}

	for _, container := range pod.Spec.Containers {
		if container.Name == hServerContainerName {
			for _, p := range container.Ports {
				if p.Name == "port" {
					return p.ContainerPort
				}
			}
		}
	}
	return hServerPort.ContainerPort
}
