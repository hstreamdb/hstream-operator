package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var hStoreEnvVar = []corev1.EnvVar{
	{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	},
	{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	},
}

var hStorePorts = []corev1.ContainerPort{
	{
		Name:          "port",
		ContainerPort: 4440,
		Protocol:      corev1.ProtocolTCP,
	},
	{
		Name:          "gossip-port",
		ContainerPort: 4441,
		Protocol:      corev1.ProtocolTCP,
	},
	{
		Name:          "admin-port",
		ContainerPort: 6440,
		Protocol:      corev1.ProtocolTCP,
	},
}

var hStoreArg = map[string]string{
	"--config-path":          internal.HStoreConfigPath + "/config.json",
	"--address":              "$(POD_IP)",
	"--name":                 "$(POD_NAME)",
	"--local-log-store-path": internal.HStoreDataPath,
	"--num-shards":           "1",
}

type addHStore struct{}

func (a addHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add hstore")

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

		logger.Info("Create hstore")
		if err = r.Client.Create(ctx, &sts); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}
	if !isHashChanged(&existingSts.ObjectMeta, &sts.ObjectMeta) {
		return nil
	}

	logger.Info("Update hstore")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingHStore", "")

	existingSts.Annotations = sts.Annotations
	existingSts.Labels = sts.Labels
	existingSts.Spec = sts.Spec
	if err = r.Update(ctx, existingSts); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addHStore) getSts(hdb *appsv1alpha1.HStreamDB) appsv1.StatefulSet {
	podTemplate := a.getPodTemplate(hdb)
	pvcs := a.getPVC(hdb)

	sts := internal.GetStatefulSet(hdb, &hdb.Spec.HStore, &podTemplate, appsv1alpha1.ComponentTypeHStore)
	sts.Spec.VolumeClaimTemplates = pvcs
	return sts
}

func (a addHStore) getPodTemplate(hdb *appsv1alpha1.HStreamDB) corev1.PodTemplateSpec {
	hStore := hdb.Spec.HStore
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, appsv1alpha1.ComponentTypeHStore),
		Spec: corev1.PodSpec{
			Affinity:        hStore.Affinity,
			Tolerations:     hStore.Tolerations,
			NodeName:        hStore.NodeName,
			NodeSelector:    hStore.NodeSelector,
			SecurityContext: hStore.PodSecurityContext,
			InitContainers:  hStore.InitContainers,
			Containers:      a.getContainer(hdb),
			Volumes:         append(hStore.Volumes, a.getVolumes(hdb)...),
		},
	}

	podTemplate.Name = appsv1alpha1.ComponentTypeHStore.GetResName(hdb.Name)
	return podTemplate
}

func (a addHStore) getContainer(hdb *appsv1alpha1.HStreamDB) []corev1.Container {
	hStore := &hdb.Spec.HStore
	container := corev1.Container{
		Image:           hdb.Spec.Image,
		ImagePullPolicy: hdb.Spec.ImagePullPolicy,
	}

	structAssign(&container, &hStore.Container)
	extendEnv(&container, hStoreEnvVar)

	if container.Name == "" {
		container.Name = string(appsv1alpha1.ComponentTypeHStore)
	}

	if len(container.Command) == 0 {
		container.Command = []string{"/usr/local/bin/logdeviced"}
	}

	args := make(map[string]string)
	for k, v := range hStoreArg {
		args[k] = v
	}
	args["--num-shards"] = strconv.Itoa(int(*hdb.Spec.Config.NShards))
	args, _ = extendArg(&container, args)

	container.Ports = extendPorts(args, container.Ports, hStorePorts)

	internal.ConfigMaps.Visit(func(m internal.ConfigMap) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      m.MountName,
			MountPath: m.MountPath,
			ReadOnly:  true,
		})
	})

	for i := int32(0); i < *hdb.Spec.Config.NShards; i++ {
		container.VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      internal.GetPvcName(hdb),
				SubPath:   fmt.Sprintf("shard%d", i),
				MountPath: fmt.Sprintf("%s/shard%d", internal.HStoreDataPath, i),
			})
	}
	return append([]corev1.Container{container}, hStore.SidecarContainers...)
}

func (a addHStore) getVolumes(hdb *appsv1alpha1.HStreamDB) (volumes []corev1.Volume) {
	volumes = make([]corev1.Volume, 0)
	internal.ConfigMaps.Visit(func(m internal.ConfigMap) {
		volumes = append(volumes, corev1.Volume{
			Name: m.MountName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: internal.GetResNameOnPanic(hdb, m.MapNameSuffix)},
					Items: []corev1.KeyToPath{
						{
							Key:  m.MapKey,
							Path: m.MapPath,
						},
					},
				}},
		})
	})

	if !usePVC(hdb) {
		volumes = append(volumes, corev1.Volume{
			Name: internal.GetPvcName(hdb),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	return
}

func (a addHStore) getPVC(hdb *appsv1alpha1.HStreamDB) (pvc []corev1.PersistentVolumeClaim) {
	if usePVC(hdb) {
		return []corev1.PersistentVolumeClaim{internal.GetPvc(hdb)}
	}
	return nil
}
