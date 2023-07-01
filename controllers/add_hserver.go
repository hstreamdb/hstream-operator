package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var hServerEnvVar = []corev1.EnvVar{
	{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	},
}

var hServerPort = corev1.ContainerPort{
	Name:          "port",
	ContainerPort: 6570,
	Protocol:      corev1.ProtocolTCP,
}

var hServerInternalPort = corev1.ContainerPort{
	Name:          "internal-port",
	ContainerPort: 6571,
	Protocol:      corev1.ProtocolTCP,
}

type addHServer struct{}

func (a addHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add hserver")

	sts := a.getSts(hdb)

	existingSts := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(&sts), existingSts)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}
		if err = ctrl.SetControllerReference(hdb, &sts, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		logger.Info("Create hServer")
		if err = r.Client.Create(ctx, &sts); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}
	if !isHashChanged(&existingSts.ObjectMeta, &sts.ObjectMeta) {
		return nil
	}

	logger.Info("Update hServer")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingHServer", "")

	existingSts.Annotations = sts.Annotations
	existingSts.Labels = sts.Labels
	existingSts.Spec = sts.Spec
	if err = r.Update(ctx, existingSts); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addHServer) getSts(hdb *hapi.HStreamDB) appsv1.StatefulSet {
	podTemplate := a.getPodTemplate(hdb)
	sts := internal.GetStatefulSet(hdb, &hdb.Spec.HServer, &podTemplate, hapi.ComponentTypeHServer)
	return sts
}

func (a addHServer) getPodTemplate(hdb *hapi.HStreamDB) corev1.PodTemplateSpec {
	hServer := &hdb.Spec.HServer

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHServer),
		Spec: corev1.PodSpec{
			Affinity:        hServer.Affinity,
			Tolerations:     hServer.Tolerations,
			NodeName:        hServer.NodeName,
			NodeSelector:    hServer.NodeSelector,
			SecurityContext: hServer.PodSecurityContext,
			InitContainers:  hServer.InitContainers,
			Containers:      a.getContainer(hdb),
			Volumes:         append(hServer.Volumes, a.getVolumes(hdb)...),
		},
	}

	podTemplate.Name = hapi.ComponentTypeHServer.GetResName(hdb.Name)
	return podTemplate
}

func (a addHServer) getContainer(hdb *hapi.HStreamDB) []corev1.Container {
	hServer := &hdb.Spec.HServer
	container := corev1.Container{
		Image:           hdb.Spec.HServer.Image,
		ImagePullPolicy: hdb.Spec.HServer.ImagePullPolicy,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromString("port"),
				},
			},
			FailureThreshold: 30,
			PeriodSeconds:    1,
		},
	}

	structAssign(&container, &hServer.Container)
	container.Env = extendEnvs(container.Env, hServerEnvVar...)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeHServer)
	}

	container.Command, container.Args, container.Ports = a.defaultCommandArgsAndPorts(hdb)

	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{Name: m.MountName, MountPath: m.MountPath},
	)
	return append([]corev1.Container{container}, hServer.SidecarContainers...)
}

func (a addHServer) getVolumes(hdb *hapi.HStreamDB) (volumes []corev1.Volume) {
	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	volumes = []corev1.Volume{internal.GetVolume(hdb, m)}
	return
}

func (a addHServer) defaultCommandArgsAndPorts(hdb *hapi.HStreamDB) (command, args []string, ports []corev1.ContainerPort) {
	if len(hdb.Spec.HServer.Container.Command) > 0 {
		return hdb.Spec.HServer.Container.Command, hdb.Spec.HServer.Container.Args, coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(hdb.Spec.HServer.Container.Ports, hServerPort, hServerInternalPort))
	}
	if len(hdb.Spec.HServer.Container.Args) > 0 {
		if strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "bash") ||
			strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "sh") ||
			strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "-c") {
			return hdb.Spec.HServer.Container.Command, hdb.Spec.HServer.Container.Args, coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(hdb.Spec.HServer.Container.Ports, hServerPort, hServerInternalPort))
		}
	}

	command = []string{"bash", "-c"}
	preArgs := []string{"/usr/local/bin/hstream-server"}
	ports = hdb.Spec.HServer.Container.Ports

	flags := internal.FlagSet{}
	if len(hdb.Spec.HServer.Container.Args) > 0 {
		_ = flags.Parse(hdb.Spec.HServer.Container.Args)
	}
	if _, ok := flags.Flags()["--config-path"]; !ok {
		args = append(args, "--config-path", "/etc/hstream/config.yaml")
	}
	if _, ok := flags.Flags()["--bind-address"]; !ok {
		args = append(args, "--bind-address", "0.0.0.0")
	}
	if _, ok := flags.Flags()["--advertised-address"]; !ok {
		args = append(args, "--advertised-address", "$(POD_IP)")
	}
	if _, ok := flags.Flags()["--store-config"]; !ok {
		args = append(args, "--store-config", "/etc/logdevice/config.json")
	}
	if _, ok := flags.Flags()["--store-admin-host"]; !ok {
		args = append(args, "--store-admin-host", internal.GetService(hdb, hapi.ComponentTypeAdminServer).Name+"."+hdb.GetNamespace())
	}
	if _, ok := flags.Flags()["--metastore-uri"]; !ok {
		hmeta, _ := getHMetaAddr(hdb)
		args = append(args, "--metastore-uri", "rq://"+hmeta)
	}
	if _, ok := flags.Flags()["--server-id"]; !ok {
		args = append(args, "--server-id", "$(hostname | grep -o '[0-9]*$')")
	}
	if _, ok := flags.Flags()["--seed-nodes"]; !ok {
		hServerSvc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer)
		seedNodes := make([]string, hdb.Spec.HServer.Replicas)

		f := internal.FlagSet{}
		_ = f.Parse(args)
		internalPort := f.Flags()["--internal-port"]

		for i := int32(0); i < hdb.Spec.HServer.Replicas; i++ {
			// ep. hstreamdb-sample-hserver-0.hstreamdb-sample-internal-hserver.default:6571
			seedNodes[i] = fmt.Sprintf("%s-%d.%s.%s:%s",
				hapi.ComponentTypeHServer.GetResName(hdb.Name),
				i,
				hServerSvc.Name,
				hServerSvc.Namespace,
				internalPort,
			)
		}
	}
	if _, ok := flags.Flags()["--port"]; !ok {
		args = append(args, "--port", strconv.Itoa(int(hServerPort.ContainerPort)))
		ports = coverPortsFromArgs(args, extendPorts(ports, hServerPort))
	} else {
		ports = coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(ports, hServerPort))
	}
	if _, ok := flags.Flags()["--internal-port"]; !ok {
		args = append(args, "--internal-port", strconv.Itoa(int(hServerInternalPort.ContainerPort)))
		ports = coverPortsFromArgs(args, extendPorts(ports, hServerInternalPort))
	} else {
		ports = coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(ports, hServerInternalPort))
	}

	args = append(preArgs, args...)
	args = append(args, hdb.Spec.HServer.Container.Args...)
	return command, []string{strings.Join(args, " ")}, ports
}
