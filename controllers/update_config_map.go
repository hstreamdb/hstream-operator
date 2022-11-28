package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	jsoniter "github.com/json-iterator/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type updateConfigMap struct {
}

func (u updateConfigMap) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	configMap, err := u.getLogDeviceConfigMap(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	if err = u.createOrUpdateConfigMap(ctx, r, hdb, &configMap); err != nil {
		return &requeue{curError: err}
	}

	nshardsMap, err := u.getNShardsMap(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	if err = u.createOrUpdateConfigMap(ctx, r, hdb, &nshardsMap); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (u updateConfigMap) getLogDeviceConfigMap(hdb *appsv1alpha1.HStreamDB) (configMap corev1.ConfigMap, err error) {
	config, err := internal.ParseLogDeviceConfig(hdb.Spec.Config.LogDeviceConfig.Raw)
	if err != nil {
		err = fmt.Errorf("invalid log device config: %w", err)
		return
	}

	defConfig := internal.GetLogDeviceConfig()
	for key, value := range defConfig {
		if _, ok := config[key]; !ok {
			config[key] = value
		}
	}

	cm, has := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	if !has {
		err = fmt.Errorf("no such config map %s", internal.LogDeviceConfig)
		return
	}

	file, _ := json.MarshalToString(config)
	configMap.Data = map[string]string{
		// config.json: file
		cm.MapKey: file,
	}
	configMap.Name = internal.GetResNameOnPanic(hdb, cm.MapNameSuffix)
	configMap.Namespace = hdb.GetNamespace()
	return
}

func (u updateConfigMap) getNShardsMap(hdb *appsv1alpha1.HStreamDB) (configMap corev1.ConfigMap, err error) {
	cm, has := internal.ConfigMaps.Get(internal.NShardsConfig)
	if !has {
		err = fmt.Errorf("no such config map %s", internal.NShardsConfig)
		return
	}

	var nshards string
	if hdb.Spec.Config.NShards == nil {
		nshards = "1"
	} else {
		nshards = strconv.Itoa(int(*hdb.Spec.Config.NShards))
	}

	configMap.Data = map[string]string{
		cm.MapKey: nshards,
	}
	configMap.Name = internal.GetResNameOnPanic(hdb, "nshards")
	configMap.Namespace = hdb.GetNamespace()
	return
}

func (u updateConfigMap) createOrUpdateConfigMap(ctx context.Context, r *HStreamDBReconciler,
	hdb *appsv1alpha1.HStreamDB, configMap *corev1.ConfigMap) (err error) {

	logger := log.WithValues("namespace", configMap.Namespace,
		"instance", hdb.Name, "name", configMap.Name, "reconciler", "UpdateConfigMap")

	existing := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKeyFromObject(configMap), existing)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(hdb, configMap, r.Scheme); err != nil {
				return
			}
			logger.Info("Creating config map", "name", configMap.Name)
			return r.Create(ctx, configMap)
		}
		return
	}

	needUpdate := !equality.Semantic.DeepEqual(existing.Data, configMap.Data)
	if !equality.Semantic.DeepEqual(existing.BinaryData, configMap.BinaryData) {
		needUpdate = true
	}

	if !needUpdate {
		return nil
	}

	logger.Info("Updating config map")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingConfigMap", "")

	existing.Data = configMap.Data
	existing.BinaryData = configMap.BinaryData
	return r.Update(ctx, existing)
}
