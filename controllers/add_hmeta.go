package controllers

import (
	"context"
	"fmt"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var hmetaPorts = []corev1.ContainerPort{
	{
		Name:          "port",
		ContainerPort: 4001,
		Protocol:      corev1.ProtocolTCP,
	},
}

var hmetaArg = map[string]string{
	"--disco-mode":    "dns",
	"--join-interval": "1s",
	"--join-attempts": "120",
	//"--disco-config":          "{\"name\":\"rqlite-svc-internal\"}",
	//"--bootstrap-expect": 1,
}

type addHMeta struct{}

func (a addHMeta) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add hmeta")

	if hdb.Spec.ExternalHMeta != nil {
		logger.Info("Using external hmeta cluster")
		return nil
	}

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

		logger.Info("Create hmeta")
		if err = r.Client.Create(ctx, &sts); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}
	if !isHashChanged(&existingSts.ObjectMeta, &sts.ObjectMeta) {
		return nil
	}

	logger.Info("Update hmeta")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingHMeta", "")

	existingSts.Annotations = sts.Annotations
	existingSts.Labels = sts.Labels
	existingSts.Spec.Replicas = sts.Spec.Replicas
	existingSts.Spec.Template = sts.Spec.Template
	existingSts.Spec.UpdateStrategy = sts.Spec.UpdateStrategy
	existingSts.Spec.MinReadySeconds = sts.Spec.MinReadySeconds
	if err = r.Update(ctx, existingSts); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addHMeta) getSts(hdb *hapi.HStreamDB) appsv1.StatefulSet {
	podTemplate := a.getPodTemplate(hdb)
	pvcs := a.getPVC(hdb)

	sts := internal.GetStatefulSet(hdb, &hdb.Spec.HMeta, &podTemplate, hapi.ComponentTypeHMeta)
	sts.Spec.VolumeClaimTemplates = pvcs
	return sts
}

func (a addHMeta) getPodTemplate(hdb *hapi.HStreamDB) corev1.PodTemplateSpec {
	hmeta := hdb.Spec.HMeta
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHMeta),
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &[]int64{5}[0],
			Affinity:                      hmeta.Affinity,
			Tolerations:                   hmeta.Tolerations,
			NodeName:                      hmeta.NodeName,
			NodeSelector:                  hmeta.NodeSelector,
			SecurityContext:               hmeta.PodSecurityContext,
			InitContainers:                hmeta.InitContainers,
			Containers:                    a.getContainer(hdb),
			Volumes:                       append(hmeta.Volumes, a.getVolumes(hdb)...),
		},
	}

	podTemplate.Name = hapi.ComponentTypeHMeta.GetResName(hdb.Name)
	return podTemplate
}

func (a addHMeta) getContainer(hdb *hapi.HStreamDB) []corev1.Container {
	hmeta := &hdb.Spec.HMeta
	container := corev1.Container{
		Image:           hdb.Spec.HMeta.Image,
		ImagePullPolicy: hdb.Spec.HMeta.ImagePullPolicy,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readyz",
					Port:   intstr.FromString("port"),
					Scheme: "HTTP",
				},
			},
			PeriodSeconds:       5,
			TimeoutSeconds:      2,
			InitialDelaySeconds: 2,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readyz?noleader",
					Port:   intstr.FromString("port"),
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 2,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
		},
	}

	structAssign(&container, &hmeta.Container)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeHMeta)
	}

	args := make(map[string]string)
	for k, v := range hmetaArg {
		args[k] = v
	}
	args["--bootstrap-expect"] = strconv.Itoa(int(hmeta.Replicas))
	svc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHMeta)
	args["--disco-config"] = fmt.Sprintf(`{"name":"%s"}`, svc.Name)

	// parsedArgs don't contain prefix "-" or "--"
	parsedArgs, _ := extendArg(&container, args)
	container.Ports = getHMetaContainerPorts(&hmeta.Container, parsedArgs)

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      internal.GetPvcName(hdb, hdb.Spec.HMeta.VolumeClaimTemplate),
			MountPath: internal.HMetaDataPath,
		})
	return append([]corev1.Container{container}, hmeta.SidecarContainers...)
}

func (a addHMeta) getVolumes(hdb *hapi.HStreamDB) (volumes []corev1.Volume) {
	volumes = make([]corev1.Volume, 0)

	// add an emptyDir volume if the pvc is null
	if hdb.Spec.HMeta.VolumeClaimTemplate == nil {
		volumes = append(volumes, corev1.Volume{
			Name: internal.GetPvcName(hdb, hdb.Spec.HMeta.VolumeClaimTemplate),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	return
}

func (a addHMeta) getPVC(hdb *hapi.HStreamDB) (pvc []corev1.PersistentVolumeClaim) {
	if hdb.Spec.HMeta.VolumeClaimTemplate != nil {
		return []corev1.PersistentVolumeClaim{
			internal.GetPvc(hdb, hdb.Spec.HMeta.VolumeClaimTemplate, hapi.ComponentTypeHMeta),
		}
	}
	return nil
}
