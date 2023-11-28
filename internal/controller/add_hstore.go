package controller

import (
	"context"
	"fmt"
	"strconv"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var hStoreArgs = []string{
	"--config-path", internal.HStoreConfigPath + "/config.json",
	"--address", "$(POD_IP)",
	"--name", "$(POD_NAME)",
	"--local-log-store-path", internal.HStoreDataPath,
}

type addHStore struct{}

func (a addHStore) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add HStore")

	// modify nShard will impact the data storage path of HStore,
	// so we need to get old nshards config from the existing config map
	nShard := a.getNShardFromExistingConfigMap(ctx, r, hdb)

	sts := a.getSts(hdb, nShard)

	existingSts := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(&sts), existingSts)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}
		if err = ctrl.SetControllerReference(hdb, &sts, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		logger.Info("Create HStore")
		if err = r.Client.Create(ctx, &sts); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}

	if !isHashChanged(&existingSts.ObjectMeta, &sts.ObjectMeta) {
		return nil
	}

	oldReplicas := existingSts.Spec.Replicas
	newReplicas := sts.Spec.Replicas

	if getRecommendedLogReplicaAcross(*newReplicas) > *newReplicas {
		logger.Error(fmt.Errorf("--metadata-replicate-across should be less than or equal to %d", *newReplicas),
			"invalid replicas", "replicas", *newReplicas)

		return nil
	}

	sts.Annotations[hapi.OldReplicas] = strconv.Itoa(int(*oldReplicas))
	sts.Annotations[hapi.NewReplicas] = strconv.Itoa(int(*newReplicas))

	existingSts.Labels = sts.Labels
	existingSts.Annotations = sts.Annotations
	existingSts.Spec.Template = sts.Spec.Template
	existingSts.Spec.UpdateStrategy = sts.Spec.UpdateStrategy
	existingSts.Spec.MinReadySeconds = sts.Spec.MinReadySeconds

	logger.Info("Updating HStore StatefulSet", "StatefulSet", sts.Name)
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingHStore", fmt.Sprintf("Updating HStore StatefulSet %s", sts.Name))

	if err = r.Update(ctx, existingSts); err != nil {
		return &requeue{curError: err}
	}

	// If HStore is being scaled up/down, set the HStoreUpdating condition to true.
	if *oldReplicas != *newReplicas {
		if *oldReplicas < *newReplicas {
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.HStoreUpdating,
				Status:  metav1.ConditionTrue,
				Reason:  hapi.HStoreScalingUp,
				Message: fmt.Sprintf("HStore is scaling up, old replicas: %d, new replicas: %d", *oldReplicas, *newReplicas),
			})
		} else {
			hdb.SetCondition(metav1.Condition{
				Type:    hapi.HStoreUpdating,
				Status:  metav1.ConditionTrue,
				Reason:  hapi.HStoreScalingDown,
				Message: fmt.Sprintf("HStore is scaling down, old replicas: %d, new replicas: %d", *oldReplicas, *newReplicas),
			})
		}

		if err = r.Status().Update(ctx, hdb); err != nil {
			return &requeue{curError: fmt.Errorf("failed to update HStore status: %w", err)}
		}
	}

	return nil
}

func (a addHStore) getSts(hdb *hapi.HStreamDB, nShard int32) appsv1.StatefulSet {
	podTemplate := a.getPodTemplate(hdb, nShard)
	pvcs := a.getPVC(hdb)

	sts := internal.GetStatefulSet(hdb, &hdb.Spec.HStore, &podTemplate, hapi.ComponentTypeHStore)
	sts.Spec.VolumeClaimTemplates = pvcs
	return sts
}

func (a addHStore) getPodTemplate(hdb *hapi.HStreamDB, nShard int32) corev1.PodTemplateSpec {
	hStore := hdb.Spec.HStore
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHStore),
		Spec: corev1.PodSpec{
			Affinity:        hStore.Affinity,
			Tolerations:     hStore.Tolerations,
			NodeName:        hStore.NodeName,
			NodeSelector:    hStore.NodeSelector,
			SecurityContext: hStore.PodSecurityContext,
			InitContainers:  hStore.InitContainers,
			Containers:      a.getContainer(hdb, nShard),
			Volumes:         append(hStore.Volumes, a.getVolumes(hdb)...),
		},
	}

	podTemplate.Name = hapi.ComponentTypeHStore.GetResName(hdb.Name)
	return podTemplate
}

func (a addHStore) getContainer(hdb *hapi.HStreamDB, nShard int32) []corev1.Container {
	hStore := &hdb.Spec.HStore
	container := corev1.Container{
		Image:           hdb.Spec.HStore.Image,
		ImagePullPolicy: hdb.Spec.HStore.ImagePullPolicy,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromString("admin-port"),
				},
			},
			FailureThreshold: 30,
			PeriodSeconds:    1,
		},
	}

	structAssign(&container, &hStore.Container)

	container.Env = extendEnvs(container.Env, constants.DefaultHStoreEnv...)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeHStore)
	}

	if len(container.Command) == 0 {
		container.Command = []string{"/usr/local/bin/logdeviced"}
	}

	args := hStoreArgs
	args = append(args,
		"--server-id", "$(hostname | grep -o '[0-9]*$')",
		"--num-shards", strconv.Itoa(int(nShard)),
	)

	container.Args, _ = extendArgs(container.Args, args...)
	container.Ports = coverPortsFromArgs(container.Args, extendPorts(container.Ports, constants.DefaultHStorePorts...))

	internal.ConfigMaps.Visit(func(m internal.ConfigMap) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      m.MountName,
			MountPath: m.MountPath,
			ReadOnly:  true,
		})
	})

	for i := int32(0); i < nShard; i++ {
		container.VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      internal.GetPvcName(hdb, hdb.Spec.HStore.VolumeClaimTemplate),
				SubPath:   fmt.Sprintf("shard%d", i),
				MountPath: fmt.Sprintf("%s/shard%d", internal.HStoreDataPath, i),
			})
	}
	return append([]corev1.Container{container}, hStore.SidecarContainers...)
}

func (a addHStore) getVolumes(hdb *hapi.HStreamDB) (volumes []corev1.Volume) {
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

	// add an emptyDir volume if the pvc is null
	if hdb.Spec.HStore.VolumeClaimTemplate == nil {
		volumes = append(volumes, corev1.Volume{
			Name: internal.GetPvcName(hdb, hdb.Spec.HStore.VolumeClaimTemplate),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	return
}

func (a addHStore) getPVC(hdb *hapi.HStreamDB) (pvc []corev1.PersistentVolumeClaim) {
	if hdb.Spec.HStore.VolumeClaimTemplate != nil {
		return []corev1.PersistentVolumeClaim{
			internal.GetPvc(hdb, hdb.Spec.HStore.VolumeClaimTemplate, hapi.ComponentTypeHStore),
		}
	}
	return nil
}

// return nShard config from existing configmap, or new config if not found
func (a addHStore) getNShardFromExistingConfigMap(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) int32 {
	nShard := hdb.Spec.Config.NShards

	var existingConfigMap corev1.ConfigMap
	nShardConfigMap, _ := getNShardsMap(hdb)
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(&nShardConfigMap), &existingConfigMap)
	if err == nil {
		for _, v := range existingConfigMap.Data {
			if num, err := strconv.Atoi(v); err == nil {
				nShard = int32(num)
			}
		}
	}
	return nShard
}
