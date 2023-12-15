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
	"strconv"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type LogDeviceConfigReconciler struct{}

func (lc LogDeviceConfigReconciler) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "LogDeviceConfigReconciler")

	var logDeviceConfigMap corev1.ConfigMap
	logDeviceConfigMapNamespacedName := utils.GetLogDeviceConfigMapNamespacedName(hdb)
	if err := r.Get(ctx, logDeviceConfigMapNamespacedName, &logDeviceConfigMap); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}

		hmetaAddr, _ := utils.GetHMetaAddr(hdb)
		logDeviceConfig, _ := utils.GetLogDeviceConfig(hdb.Spec.HStore.Replicas, hmetaAddr, hdb.Spec.Config.LogDeviceConfig.Raw)

		logDeviceConfigMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      logDeviceConfigMapNamespacedName.Name,
				Namespace: logDeviceConfigMapNamespacedName.Namespace,
			},
			Data: map[string]string{
				utils.LogDeviceConfigKey: logDeviceConfig,
			},
		}

		if err = ctrl.SetControllerReference(hdb, &logDeviceConfigMap, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		if err = r.Create(ctx, &logDeviceConfigMap); err != nil {
			logger.Error(err, "failed to create ConfigMap for LogDevice config",
				"ConfigMap", logDeviceConfigMap.Name)

			return &requeue{curError: err}
		}
	}

	var nShardsConfigMap corev1.ConfigMap
	nShardsConfigMapNamespacedName := utils.GetNShardsConfigMapNamespacedName(hdb)
	if err := r.Get(ctx, nShardsConfigMapNamespacedName, &nShardsConfigMap); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil
		}

		nShardsConfigMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nShardsConfigMapNamespacedName.Name,
				Namespace: nShardsConfigMapNamespacedName.Namespace,
			},
			Data: map[string]string{
				utils.NShardsConfigKey: strconv.Itoa(utils.GetMinNShards(hdb)),
			},
		}

		if err = ctrl.SetControllerReference(hdb, &nShardsConfigMap, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		if err = r.Create(ctx, &nShardsConfigMap); err != nil {
			logger.Error(err, "failed to create ConfigMap for nShards config",
				"ConfigMap", nShardsConfigMap.Name)

			return &requeue{curError: err}
		}
	}

	// TODO: updating the configuration is currently unavailable as some configuration items cannot be modified.

	return nil
}
