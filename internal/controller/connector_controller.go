/*
Copyright 2023.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1beta1 "github.com/hstreamdb/hstream-operator/api/v1beta1"
)

const (
	connectorfinalizerName = "connectortemplate.hstream.io/finalizer"
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

	var connector appsv1beta1.Connector
	if err := r.Get(ctx, req.NamespacedName, &connector); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "fail to fetch Connector")
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configMapName := appsv1beta1.GenConnectorConfigMapName(connector.Name, false)
	configMapNamespacedName := types.NamespacedName{
		Namespace: connector.Namespace,
		Name:      configMapName,
	}

	var connectorConfigMap corev1.ConfigMap
	if err := r.Get(ctx, configMapNamespacedName, &connectorConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			configs, err := r.mergePatchesIntoConfigs(ctx, connector)
			if err != nil {
				log.Error(err, "fail to merge patches into config")

				return ctrl.Result{}, err
			}

			for index, config := range *configs {
				configWithService := map[string]interface{}{
					"connector": config,
					"hstream": map[string]string{
						"serviceUrl": "hstream://" + connector.Spec.HServerEndpoint,
					},
				}

				configJson, _ := json.Marshal(configWithService)

				err = r.Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: connector.Namespace,
						Name:      configMapName,
					},
					Data: map[string]string{
						"config.json": string(configJson),
					},
				})
				if err != nil {
					log.Error(err, "fail to create ConfigMap for Connector",
						"Connector", connector.Name,
						"ConfigMap", configMapName,
					)

					return ctrl.Result{}, err
				}

				err = r.createConnectorDeployment(ctx, connector, connector.Spec.Streams[index], configMapName)
				if err != nil {
					log.Error(err, "fail to create Deployment for Connector",
						"Connector", connector.Name,
						"Deployment", appsv1beta1.GenConnectorDeploymentName(connector.Name, connector.Spec.Streams[index]),
					)

					return ctrl.Result{}, err
				}
			}
		} else {
			log.Error(err, "fail to fetch ConfigMap which generated by Connector",
				"Connector", connector.Name,
				"ConfigMap", configMapName,
			)

			return ctrl.Result{}, err
		}
	}

	if connector.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&connector, connectorfinalizerName) {
			controllerutil.AddFinalizer(&connector, connectorfinalizerName)

			if err := r.Update(ctx, &connector); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&connector, connectorfinalizerName) {
			// Clear generated ConfigMap if Connector is being deleted.
			if err := appsv1beta1.DeleteAssociatedConfigMap(ctx, r.Client, configMapNamespacedName); err != nil {
				log.Error(err, "fail to delete ConfigMap which generated by Connector",
					"Connector", connector.Name,
					"ConfigMap", configMapName,
				)

				return ctrl.Result{}, err
			}

			if err := r.deleteAssociatedDeployments(ctx, connector); err != nil {
				log.Error(err, "fail to delete Deployment which generated by Connector",
					"Connector", connector.Name,
				)

				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&connector, connectorfinalizerName)
			if err := r.Update(ctx, &connector); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.Connector{}).
		Complete(r)
}

func (r *ConnectorReconciler) mergePatchesIntoConfigs(ctx context.Context, connector appsv1beta1.Connector) (*[]map[string]interface{}, error) {
	templateConfigMapName := appsv1beta1.GenConnectorConfigMapName(connector.Spec.TemplateName, true)
	templateConfigMapNamespacedName := types.NamespacedName{
		Namespace: connector.Namespace,
		Name:      templateConfigMapName,
	}

	var templateConfigMap corev1.ConfigMap
	if err := r.Get(ctx, templateConfigMapNamespacedName, &templateConfigMap); err != nil {
		log.Error(err, "fail to fetch ConfigMap which stores ConnectorTemplate config")

		return nil, err
	}

	var templateConfig map[string]interface{}
	err := json.Unmarshal([]byte(templateConfigMap.Data["config"]), &templateConfig)
	if err != nil {
		log.Error(err, "fail to unmarshal ConnectorTemplate config")

		return nil, err
	}

	var configs []map[string]interface{}

	for _, stream := range connector.Spec.Streams {
		config := make(map[string]interface{})
		for k, v := range templateConfig {
			config[k] = v
		}

		config["stream"] = stream

		var patches map[string]map[string]interface{}
		if connector.Spec.Patches != nil {
			json.Unmarshal(connector.Spec.Patches, &patches)
		}

		if connector.Spec.Patches != nil {
			if val, ok := patches[stream]; ok {
				for k, v := range val {
					config[k] = v
				}
			}
		}

		configs = append(configs, config)
	}

	return &configs, nil
}

func (r *ConnectorReconciler) createConnectorDeployment(ctx context.Context, connector appsv1beta1.Connector, stream, configMapName string) error {
	name := appsv1beta1.GenConnectorDeploymentName(connector.Name, stream)

	return r.Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: connector.Namespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "hstream-io-connector",
					"connector": connector.Name,
					"stream":    stream,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "hstream-io-connector",
						"connector": connector.Name,
						"stream":    stream,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  connector.Name,
							Image: appsv1beta1.ConnectorImageMap[connector.Spec.Type],
							Args: []string{
								"run",
								"--config /data/config/config.json",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: appsv1beta1.ConnectorContainerPortMap[connector.Spec.Type],
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configMapName,
									MountPath: "/data/config",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
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
					},
				},
			},
		},
	})
}

func (r *ConnectorReconciler) deleteAssociatedDeployments(ctx context.Context, connector appsv1beta1.Connector) error {
	for _, stream := range connector.Spec.Streams {
		name := appsv1beta1.GenConnectorDeploymentName(connector.Name, stream)
		namespacedName := types.NamespacedName{
			Namespace: connector.Namespace,
			Name:      name,
		}

		var deployment appsv1.Deployment
		if err := r.Get(ctx, namespacedName, &deployment); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "fail to fetch Deployment which generated by Connector",
					"Connector", connector.Name,
					"Deployment", name,
				)

				return err
			}
		} else {
			if err := r.Delete(ctx, &deployment); err != nil {
				log.Error(err, "fail to delete Deployment which generated by Connector",
					"Connector", connector.Name,
					"Deployment", name,
				)

				return err
			}
		}
	}

	return nil
}
