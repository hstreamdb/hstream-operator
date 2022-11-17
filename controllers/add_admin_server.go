package controllers

import (
	"context"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

const (
	adminServerConfigPath = "/etc/logdevice"
)

var adminServerArg = map[string]string{
	"--config-path":                                  adminServerConfigPath + "/config.json",
	"--enable-maintenance-manager":                   "",
	"--maintenance-log-snapshotting":                 "",
	"--enable-safety-check-periodic-metadata-update": "",
}

var adminServerPorts = []corev1.ContainerPort{
	{
		Name:          "port",
		ContainerPort: 6440,
		Protocol:      corev1.ProtocolTCP,
	},
}

type addAdminServer struct{}

func (a addAdminServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add admin server")

	deploy := a.getDeployment(hdb)

	existingDeploy := &appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(&deploy), existingDeploy)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}
		if err = ctrl.SetControllerReference(hdb, &deploy, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		logger.Info("Create admin server")
		if err = r.Client.Create(ctx, &deploy); err != nil {
			return &requeue{curError: err}
		}
		return nil
	} else if !isHashChanged(&existingDeploy.ObjectMeta, &deploy.ObjectMeta) {
		return nil
	}

	logger.Info("Update admin server")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingAdminServer", "")

	existingDeploy.Annotations = deploy.Annotations
	existingDeploy.Labels = deploy.Labels
	existingDeploy.Spec = deploy.Spec
	if err = r.Update(ctx, existingDeploy); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addAdminServer) getDeployment(hdb *appsv1alpha1.HStreamDB) appsv1.Deployment {
	podTemplate := a.getPodTemplate(hdb)
	deploy := internal.GetDeployment(hdb, &hdb.Spec.AdminServer,
		&podTemplate, appsv1alpha1.ComponentTypeAdminServer)

	return deploy
}

func (a addAdminServer) getPodTemplate(hdb *appsv1alpha1.HStreamDB) corev1.PodTemplateSpec {
	adminServer := &hdb.Spec.AdminServer

	pod := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, appsv1alpha1.ComponentTypeAdminServer),
		Spec: corev1.PodSpec{
			Affinity:        adminServer.Affinity,
			Tolerations:     adminServer.Tolerations,
			NodeName:        adminServer.NodeName,
			NodeSelector:    adminServer.NodeSelector,
			SecurityContext: adminServer.PodSecurityContext,
			InitContainers:  adminServer.InitContainers,
			Containers:      a.getContainer(hdb),
			Volumes:         adminServer.Volumes,
		},
	}

	pod.Name = appsv1alpha1.ComponentTypeAdminServer.GetResName(hdb.Name)
	pod.Spec.Volumes = append(pod.Spec.Volumes, a.getVolumes(hdb)...)
	return pod
}

func (a addAdminServer) getContainer(hdb *appsv1alpha1.HStreamDB) []corev1.Container {
	adminServer := &hdb.Spec.AdminServer
	container := corev1.Container{
		Image:           hdb.Spec.Image,
		ImagePullPolicy: hdb.Spec.ImagePullPolicy,
	}

	structAssign(&container, &adminServer.Container)
	container.Ports = mergePorts(adminServerPorts, container.Ports)

	if container.Name == "" {
		container.Name = string(appsv1alpha1.ComponentTypeAdminServer)
	}

	if len(container.Command) == 0 {
		container.Command = []string{"/usr/local/bin/ld-admin-server"}
	}

	args := make(map[string]string)
	for k, v := range adminServerArg {
		args[k] = v
	}

	for _, p := range container.Ports {
		args["--"+(&p).Name] = strconv.Itoa(int((&p).ContainerPort))
	}

	extendArg(&container, args)

	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{Name: m.MountName, MountPath: m.MountPath},
	)
	return append([]corev1.Container{container}, adminServer.SidecarContainers...)
}

func (a addAdminServer) getVolumes(hdb *appsv1alpha1.HStreamDB) (volumes []corev1.Volume) {
	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	volumes = []corev1.Volume{internal.GetVolume(hdb, m)}
	return
}
