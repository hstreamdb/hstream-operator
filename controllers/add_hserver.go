package controllers

import (
	"context"
	"fmt"
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

const (
	hServerStoreConfig = "/etc/logdevice"
)

var hServerArgs = []string{
	"--config-path", "/etc/hstream/config.yaml",
	"--bind-address", "0.0.0.0",
	"--advertised-address", "$(POD_IP)",
	"--store-config", hServerStoreConfig + "/config.json",
	//"--port",             "6570",
	//"--internal-port",    "6571",
	//"--seed-nodes",       "hstream-server-0.hstream-server:6571", // fill this while reconciling deployment
	//"--server-id",        "", // fill this while reconciling deployment
	//"--store-admin-host", "", // fill this while reconciling deployment
	//"--metastore-uri",    "rqlite://rqlite-svc.default:4001",
}

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
	// TODO: delete this special handle while hstream remove the seen-nodes and server-id arg
	if len(hdb.Spec.HServer.Container.Command) == 0 {
		sts.Spec.Replicas = &[]int32{1}[0]
	}
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
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromString("port"),
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
		},
	}

	structAssign(&container, &hServer.Container)
	container.Env = extendEnvs(container.Env, hServerEnvVar...)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeHServer)
	}

	if len(container.Command) == 0 {
		container.Command = []string{"/usr/local/bin/hstream-server"}

		args := hServerArgs
		// TODO: remove server-id
		args = append(args, "--server-id", "100")
		// TODO: rename "rq" to "ip"
		hmeta, _ := getHMetaAddr(hdb)
		args = append(args, "--metastore-uri", "rq://"+hmeta)
		args = append(args, "--store-admin-host", internal.GetService(hdb, hapi.ComponentTypeAdminServer).Name+"."+hdb.GetNamespace())

		container.Args, _ = extendArgs(container.Args, args...)
		container.Ports = coverPortsFromArgs(container.Args, extendPorts(container.Ports, hServerPort, hServerInternalPort))

		// TODO: remove seed nodes
		flags := internal.FlagSet{}
		_ = flags.Parse(container.Args)
		if _, ok := flags.Flags()["--seed-nodes"]; !ok {
			var internalPort int32
			for _, port := range container.Ports {
				if port.Name == "internal-port" {
					internalPort = port.ContainerPort
				}
			}

			hServerSvc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer)
			seedNodes := make([]string, hdb.Spec.HServer.Replicas)
			for i := int32(0); i < hdb.Spec.HServer.Replicas; i++ {
				// ep. hdbName-hserver-0.svcName.namespace:6571
				seedNodes[i] = fmt.Sprintf("%s-%d.%s.%s:%d",
					hapi.ComponentTypeHServer.GetResName(hdb.Name),
					i,
					hServerSvc.Name,
					hServerSvc.Namespace,
					internalPort)
			}
			container.Args = append(container.Args, "--seed-nodes", strings.Join(seedNodes, ","))
		}
	}

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
