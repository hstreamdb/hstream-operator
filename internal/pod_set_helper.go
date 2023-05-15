package internal

import (
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/mitchellh/hashstructure/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDeployment(hdb *hapi.HStreamDB, comp *hapi.Component, podSpec *corev1.PodTemplateSpec,
	compType hapi.ComponentType) appsv1.Deployment {

	deploy := appsv1.Deployment{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
		Spec: appsv1.DeploymentSpec{
			Replicas: &comp.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSpec.GetLabels(),
			},
			Template: *podSpec,
		},
	}

	deploy.Annotations[hapi.LastSpecKey] = GetObjectHash(&deploy)

	return deploy
}

func GetStatefulSet(hdb *hapi.HStreamDB, comp *hapi.Component, podSpec *corev1.PodTemplateSpec,
	compType hapi.ComponentType) appsv1.StatefulSet {

	service := GetHeadlessService(hdb, compType)
	sts := appsv1.StatefulSet{
		ObjectMeta: GetObjectMetadata(hdb, nil, compType),
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &comp.Replicas,
			ServiceName: service.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSpec.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            *podSpec,
		},
	}

	sts.Annotations[hapi.LastSpecKey] = GetObjectHash(&sts)

	return sts
}

func GetObjectHash(obj interface{}) string {
	hash, _ := hashstructure.Hash(obj, hashstructure.FormatV2, &hashstructure.HashOptions{SlicesAsSets: true})
	return fmt.Sprint(hash)
}
