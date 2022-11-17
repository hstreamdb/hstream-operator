package internal

import (
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/mitchellh/hashstructure/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDeployment(hdb *appsv1alpha1.HStreamDB, comp *appsv1alpha1.Component, podSpec *corev1.PodTemplateSpec,
	compType appsv1alpha1.ComponentType) appsv1.Deployment {

	deploy := appsv1.Deployment{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
		Spec: appsv1.DeploymentSpec{
			Replicas: comp.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSpec.GetLabels(),
			},
			Template: *podSpec,
		},
	}

	deploy.Name = compType.GetResName(hdb.Name)
	deploy.Annotations[appsv1alpha1.LastSpecKey] = GetObjectHash(&deploy)

	return deploy
}

func GetStatefulSet(hdb *appsv1alpha1.HStreamDB, comp *appsv1alpha1.Component, podSpec *corev1.PodTemplateSpec,
	compType appsv1alpha1.ComponentType) appsv1.StatefulSet {

	service := GetHeadlessService(hdb, compType)
	sts := appsv1.StatefulSet{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
		Spec: appsv1.StatefulSetSpec{
			Replicas:    comp.Replicas,
			ServiceName: service.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSpec.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            *podSpec,
		},
	}

	sts.Name = compType.GetResName(hdb.Name)
	sts.Annotations[appsv1alpha1.LastSpecKey] = GetObjectHash(&sts)

	return sts
}

func GetObjectHash(obj interface{}) string {
	hash, _ := hashstructure.Hash(obj, hashstructure.FormatV2, &hashstructure.HashOptions{SlicesAsSets: true})
	return fmt.Sprint(hash)
}
