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
)

// ConnectorTemplateReconciler reconciles a ConnectorTemplate object
type ConnectorTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectortemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectortemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.hstream.io,resources=connectortemplates/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConnectorTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *ConnectorTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log = logf.FromContext(ctx)

	var connectorTemplate v1beta1.ConnectorTemplate
	if err := r.Get(ctx, req.NamespacedName, &connectorTemplate); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !connectorTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	configMapName := v1beta1.GenConnectorConfigMapName(req.Name, true)
	configMapNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      configMapName,
	}

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, configMapNamespacedName, &configMap); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      configMapName,
			},
			Data: map[string]string{
				"config": connectorTemplate.Spec.Config,
			},
		}

		if err := controllerutil.SetControllerReference(&connectorTemplate, &configMap, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		// Create ConfigMap to store connector config.
		err = r.Create(ctx, &configMap)

		if err != nil {
			log.Error(err, "fail to create ConfigMap for ConnectorTemplate",
				"ConnectorTemplate", connectorTemplate.Name,
				"ConfigMap", configMapName,
			)

			return ctrl.Result{}, err
		}
	}

	// TODO: handle update logic.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConnectorTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ConnectorTemplate{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
