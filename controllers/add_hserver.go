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
)

const (
	hServerStoreConfig = "/etc/logdevice"
)

var hServerArg = map[string]string{
	"--config-path":        "/etc/hstream/config.yaml",
	"--bind-address":       "0.0.0.0",
	"--advertised-address": "$(POD_IP)",
	"--store-config":       hServerStoreConfig + "/config.json",
	//"--port":             "6570",
	//"--internal-port":    "6571",
	//"--seed-nodes":       "hstream-server-0.hstream-server:6571", // fill this while reconciling deployment
	//"--server-id":        "", // fill this while reconciling deployment
	//"--metastore-uri":    "rqlite://rqlite-svc.default:4001",
	//"--store-admin-host": "", // fill this while reconciling deployment
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

var hServerPorts = []corev1.ContainerPort{
	{
		Name:          "port",
		ContainerPort: 6570,
		Protocol:      corev1.ProtocolTCP,
	},
	{
		Name:          "internal-port",
		ContainerPort: 6571,
		Protocol:      corev1.ProtocolTCP,
	},
}

type addHServer struct{}

func (a addHServer) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
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

func (a addHServer) getSts(hdb *appsv1alpha1.HStreamDB) appsv1.StatefulSet {
	podTemplate := a.getPodTemplate(hdb)
	sts := internal.GetStatefulSet(hdb, &hdb.Spec.HServer, &podTemplate, appsv1alpha1.ComponentTypeHServer)

	return sts
}

func (a addHServer) getPodTemplate(hdb *appsv1alpha1.HStreamDB) corev1.PodTemplateSpec {
	hServer := &hdb.Spec.HServer

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, appsv1alpha1.ComponentTypeHServer),
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

	podTemplate.Name = appsv1alpha1.ComponentTypeHServer.GetResName(hdb.Name)
	return podTemplate
}

func (a addHServer) getContainer(hdb *appsv1alpha1.HStreamDB) []corev1.Container {
	hServer := &hdb.Spec.HServer
	container := corev1.Container{
		Image:           hdb.Spec.Image,
		ImagePullPolicy: hdb.Spec.ImagePullPolicy,
	}

	structAssign(&container, &hServer.Container)
	extendEnv(&container, hServerEnvVar)
	container.Ports = mergePorts(hServerPorts, container.Ports)

	if container.Name == "" {
		container.Name = string(appsv1alpha1.ComponentTypeHServer)
	}

	if len(container.Command) == 0 {
		container.Command = []string{"/usr/local/bin/hstream-server"}
	}
	/*
		args := make(map[string]string)
		for k, v := range hServerArg {
			args[k] = v
		}

		// TODO:
		args["--server-id"] = xxx

		config, _ := parseLogDeviceConfig(hdb.Spec.Config.LogDeviceConfig.Raw)
		if rqlite, ok := config["rqlite"]; ok {
			if uri, ok := rqlite.(map[string]interface{}); ok {
				args["--metastore-uri"] = strings.Replace(uri["rqlite_uri"].(string), "ip", "rq", 1)
			}
		}

		adminServerSvc := internal.GetService(hdb, nil, appsv1alpha1.ComponentTypeAdminServer)
		args["--store-admin-host"] = fmt.Sprintf("%s.%s", adminServerSvc.Name, adminServerSvc.Namespace)

		for _, p := range container.Ports {
			args["--"+(&p).Name] = strconv.Itoa(int((&p).ContainerPort))
		}

		// TODO: remove seed nodes
		hServerSvc := internal.GetService(hdb, nil, appsv1alpha1.ComponentTypeHServer)
		seedNodes := make([]string, *hdb.Spec.HServer.Replicas)
		for i := int32(0); i < *hdb.Spec.HServer.Replicas; i++ {
			// ep. hdbName-hserver.svcName.namespace:6571
			seedNodes[i] = fmt.Sprintf("%s-%d.%s.%s:%s",
				appsv1alpha1.ComponentTypeHServer.GetResName(hdb.Name),
				i,
				hServerSvc.Name,
				hServerSvc.Namespace,
				args["--internal-port"])
		}
		args["--seed-nodes"] = strings.Join(seedNodes, ",")

		extendArg(&container, args)
	*/

	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{Name: m.MountName, MountPath: m.MountPath},
	)
	return append([]corev1.Container{container}, hServer.SidecarContainers...)
}

func (a addHServer) getVolumes(hdb *appsv1alpha1.HStreamDB) (volumes []corev1.Volume) {
	m, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	volumes = []corev1.Volume{internal.GetVolume(hdb, m)}
	return
}
