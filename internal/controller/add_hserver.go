package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/internal/utils"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addHServer struct{}

func (a addHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add HServer")

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
	hserver := &hdb.Spec.HServer
	container := a.getServerContainer(hdb)

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHServer),
		Spec: corev1.PodSpec{
			Affinity:        hserver.Affinity,
			Tolerations:     hserver.Tolerations,
			NodeName:        hserver.NodeName,
			NodeSelector:    hserver.NodeSelector,
			SecurityContext: hserver.PodSecurityContext,
			InitContainers:  hserver.InitContainers,
			Containers:      append([]corev1.Container{container}, hserver.SidecarContainers...),
			Volumes:         append(hserver.Volumes, utils.GetLogDeviceConfigVolume(hdb)),
		},
	}

	podTemplate.Name = hapi.ComponentTypeHServer.GetResName(hdb)
	return podTemplate
}

func (a addHServer) getServerContainer(hdb *hapi.HStreamDB) corev1.Container {
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
			InitialDelaySeconds: 10,
			FailureThreshold:    15,
			PeriodSeconds:       5,
		},
	}

	structAssign(&container, &hServer.Container)
	container.Env = extendEnvs(container.Env, constants.DefaultHServerEnv...)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeHServer)
	}

	container.Command, container.Args, container.Ports = a.defaultCommandArgsAndPorts(hdb)

	container.VolumeMounts = append(
		container.VolumeMounts,
		utils.GetLogDeviceConfigVolumeMount(hdb),
	)

	return container
}

func (a addHServer) defaultCommandArgsAndPorts(hdb *hapi.HStreamDB) (command, args []string, ports []corev1.ContainerPort) {
	if len(hdb.Spec.HServer.Container.Command) > 0 {
		return hdb.Spec.HServer.Container.Command,
			hdb.Spec.HServer.Container.Args,
			coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(hdb.Spec.HServer.Container.Ports, constants.DefaultHServerPort, constants.DefaultHServerInternalPort))
	}
	if len(hdb.Spec.HServer.Container.Args) > 0 {
		if strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "bash") ||
			strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "sh") ||
			strings.HasPrefix(hdb.Spec.HServer.Container.Args[0], "-c") {
			return hdb.Spec.HServer.Container.Command,
				hdb.Spec.HServer.Container.Args,
				coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(hdb.Spec.HServer.Container.Ports, constants.DefaultHServerPort, constants.DefaultHServerInternalPort))
		}
	}

	command = []string{"bash", "-c"}
	args = []string{"/usr/local/bin/hstream-server"}
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
		args = append(args, "--advertised-address", "$(POD_NAME)."+internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer).Name+"."+hdb.GetNamespace())
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
	if _, ok := flags.Flags()["--port"]; !ok {
		args = append(args, "--port", strconv.Itoa(int(constants.DefaultHServerPort.ContainerPort)))
		ports = coverPortsFromArgs(args, extendPorts(ports, constants.DefaultHServerPort))
	} else {
		ports = coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(ports, constants.DefaultHServerPort))
	}

	internalPortStr := strconv.Itoa(int(constants.DefaultHServerInternalPort.ContainerPort))
	if _, ok := flags.Flags()["--internal-port"]; !ok {
		args = append(args, "--internal-port", internalPortStr)
		ports = coverPortsFromArgs(args, extendPorts(ports, constants.DefaultHServerInternalPort))
	} else {
		ports = coverPortsFromArgs(hdb.Spec.HServer.Container.Args, extendPorts(ports, constants.DefaultHServerInternalPort))
	}

	if _, ok := flags.Flags()["--seed-nodes"]; !ok {
		hServerSvc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer)
		seedNodes := make([]string, hdb.Spec.HServer.Replicas)

		for i := int32(0); i < hdb.Spec.HServer.Replicas; i++ {
			// E.g. hstreamdb-sample-hserver-0.hstreamdb-sample-internal-hserver.default:6571
			seedNodes[i] = fmt.Sprintf("%s-%d.%s.%s:%s",
				hapi.ComponentTypeHServer.GetResName(hdb),
				i,
				hServerSvc.Name,
				hServerSvc.Namespace,
				internalPortStr,
			)
		}

		args = append(args, "--seed-nodes", strings.Join(seedNodes, ","))
	}

	args = append(args, hdb.Spec.HServer.Container.Args...)
	return command, []string{strings.Join(args, " ")}, ports
}
