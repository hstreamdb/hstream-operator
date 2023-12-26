/*
Copyright 2023 HStream Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/hstreamdb/hstream-operator/api/v1beta1"
	"github.com/hstreamdb/hstream-operator/pkg/connectorgen"
)

// ConnectorReconciler reconciles a Connector object
type ConnectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectors/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments,verbs=create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Connector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *ConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log = logf.FromContext(ctx)

	var connector v1beta1.Connector
	if err := r.Get(ctx, req.NamespacedName, &connector); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !connector.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// If no template is specified, use patches as arguments.
	if connector.Spec.TemplateName == nil {
		for _, stream := range connector.Spec.Streams {
			err := r.createConnectorDeployment(ctx, &connector, stream, "")
			if err != nil {
				log.Error(err, "fail to create Deployment for Connector",
					"Connector", connector.Name,
					"Deployment", v1beta1.GenConnectorDeploymentName(connector.Name, stream),
				)

				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	configMapNames := make([]string, 0, len(connector.Spec.Streams))
	for _, stream := range connector.Spec.Streams {
		configMapNames = append(configMapNames, v1beta1.GenConnectorConfigMapNameForStream(connector.Name, stream))
	}

	configs, err := r.mergePatchesIntoConfigs(ctx, log, connector)
	if err != nil {
		log.Error(err, "fail to merge connector config patches into config template")

		return ctrl.Result{}, err
	}

	for index, name := range configMapNames {
		var connectorConfigMap corev1.ConfigMap
		if err = r.Get(ctx, types.NamespacedName{
			Namespace: connector.Namespace,
			Name:      name,
		}, &connectorConfigMap); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			configWithService := map[string]interface{}{
				"connector": configs[index],
				"hstream": map[string]string{
					"serviceUrl": "hstream://" + connector.Spec.HServerEndpoint,
				},
			}

			configJson, _ := json.Marshal(configWithService)

			connectorConfigMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: connector.Namespace,
					Name:      name,
				},
				Data: map[string]string{
					"config.json": string(configJson),
				},
			}

			if err = controllerutil.SetControllerReference(&connector, &connectorConfigMap, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, &connectorConfigMap)
			if err != nil {
				log.Error(err, "fail to create ConfigMap for Connector",
					"Connector", connector.Name,
					"ConfigMap", name,
				)

				return ctrl.Result{}, err
			}

			err = r.createConnectorDeployment(ctx, &connector, connector.Spec.Streams[index], name)
			if err != nil {
				log.Error(err, "fail to create Deployment for Connector",
					"Connector", connector.Name,
					"Deployment", v1beta1.GenConnectorDeploymentName(connector.Name, connector.Spec.Streams[index]),
				)

				return ctrl.Result{}, err
			}
		}
	}

	// TODO: handle update logic.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Connector{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *ConnectorReconciler) mergePatchesIntoConfigs(ctx context.Context, logger logr.Logger, connector v1beta1.Connector) ([]map[string]interface{}, error) {
	templateConfigMapName := v1beta1.GenConnectorConfigMapName(*connector.Spec.TemplateName, true)
	templateConfigMapNamespacedName := types.NamespacedName{
		Namespace: connector.Namespace,
		Name:      templateConfigMapName,
	}

	var templateConfigMap corev1.ConfigMap
	if err := r.Get(ctx, templateConfigMapNamespacedName, &templateConfigMap); err != nil {
		logger.Error(err, "fail to fetch ConfigMap which stores ConnectorTemplate config")

		return nil, err
	}

	var templateConfig map[string]interface{}
	err := json.Unmarshal([]byte(templateConfigMap.Data["config"]), &templateConfig)
	if err != nil {
		logger.Error(err, "fail to unmarshal ConnectorTemplate config")

		return nil, err
	}

	var configs []map[string]interface{}

	for _, stream := range connector.Spec.Streams {
		config := make(map[string]interface{})
		for k, v := range templateConfig {
			config[k] = v
		}

		config["stream"] = stream

		patch, err := connector.GetPatchByStream(stream)
		if err != nil {
			logger.Error(err, "fail to get Connector patch by stream")

			return nil, err
		}

		if patch != nil {
			for k, v := range patch {
				config[k] = v
			}
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func (r *ConnectorReconciler) createConnectorDeployment(ctx context.Context, connector *v1beta1.Connector, stream string, configMapName string) error {
	name := v1beta1.GenConnectorDeploymentName(connector.Name, stream)
	containerPorts := []corev1.ContainerPort{}

	if port, ok := v1beta1.ConnectorContainerPortMap[connector.Spec.Type]; ok {
		containerPorts = []corev1.ContainerPort{
			{
				ContainerPort: port,
			},
		}
	}

	if connector.Spec.Container.Ports != nil {
		containerPorts = append(containerPorts, connector.Spec.Container.Ports...)
	}
	//nolint:staticcheck,SA1019 // this block is used to keep backward compatibility.
	if connector.Spec.ContainerPorts != nil {
		containerPorts = append(containerPorts, connector.Spec.ContainerPorts...)
	}

	connector.Spec.Container.Ports = containerPorts
	preconfiguredContainer := connectorgen.GenConnectorContainer(connector, name, stream, configMapName)
	structAssign(preconfiguredContainer, &connector.Spec.Container)

	containers := []corev1.Container{
		*preconfiguredContainer,
	}
	volumes := []corev1.Volume{}

	if connector.Spec.Type == v1beta1.SinkElaticsearch {
		containers = append(containers, connectorgen.DefaultSinkElasticsearchLogContainer(connector))
		volumes = append(
			volumes,
			[]corev1.Volume{
				{
					Name: configMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapName,
							},
						},
					},
				},
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}...,
		)
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: connector.Namespace,
			Name:      name,
			Labels: map[string]string{
				hapi.ComponentKey: v1beta1.ComponentTypeConnector,
				hapi.InstanceKey:  connector.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					hapi.ComponentKey: v1beta1.ComponentTypeConnector,
					hapi.InstanceKey:  connector.Name,
					"stream":          stream,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						hapi.ComponentKey: v1beta1.ComponentTypeConnector,
						hapi.InstanceKey:  connector.Name,
						"stream":          stream,
					},
					Annotations: getPromAnnotations(connector),
				},
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(connector, &deployment, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, &deployment)
}

func getPromAnnotations(connector *v1beta1.Connector) (annotaions map[string]string) {
	annotaions = make(map[string]string)

	for k, v := range connector.Annotations {
		if strings.HasPrefix(k, "prometheus.io") {
			annotaions[k] = v
		}
	}

	return
}
