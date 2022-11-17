package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
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

	nshardsMap, err := u.getNshardsMap(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	if err = u.createOrUpdateConfigMap(ctx, r, hdb, &nshardsMap); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (u updateConfigMap) getLogDeviceConfigMap(hdb *appsv1alpha1.HStreamDB) (configMap corev1.ConfigMap, err error) {
	config, err := parseLogDeviceConfig(hdb.Spec.Config.LogDeviceConfig.Raw)
	if err != nil {
		err = errors.New("invalid log device config")
		return
	}
	if len(config) == 0 {
		err = errors.New("missing rqlite config in LogDeviceConfig")
		return
	}

	if _, ok := config["cluster"]; !ok {
		config["cluster"] = "hstore"
	}

	v, ok := config["server_settings"]
	var m map[string]interface{}
	if !ok {
		m = map[string]interface{}{}
		config["server_settings"] = m
	} else if m, ok = v.(map[string]interface{}); !ok {
		err = errors.Errorf("invalid server_settings config")
		return
	}

	m["enable-nodes-configuration-manager"] = "true"
	m["use-nodes-configuration-manager-nodes-configuration"] = "true"
	m["enable-node-self-registration"] = "true"
	m["enable-cluster-maintenance-state-machine"] = "true"

	if v, ok = config["client_settings"]; !ok {
		m = map[string]interface{}{}
		config["client_settings"] = m
	} else if m, ok = v.(map[string]interface{}); !ok {
		err = errors.Errorf("invalid server_settings config")
		return
	}

	m["enable-nodes-configuration-manager"] = "true"
	m["use-nodes-configuration-manager-nodes-configuration"] = "true"
	m["admin-client-capabilities"] = "true"

	if v, ok = config["rqlite"]; !ok {
		m = map[string]interface{}{}
		config["rqlite"] = m
	} else if m, ok = v.(map[string]interface{}); !ok {
		err = errors.Errorf("invalid server_settings config")
		return
	}

	if _, ok = m["rqlite_uri"]; !ok {
		m["rqlite_uri"] = fmt.Sprintf("ip://rqlite-svc.default:4001")
	}

	file, _ := json.MarshalToString(config)
	configMap.Data = map[string]string{
		"config.json": file,
	}
	configMap.Name = internal.GetResNameOnPanic(hdb, "logdevice-config")
	configMap.Namespace = hdb.GetNamespace()
	return
}

func (u updateConfigMap) getNshardsMap(hdb *appsv1alpha1.HStreamDB) (configMap corev1.ConfigMap, err error) {
	var nshards string
	if hdb.Spec.Config.NShards == nil {
		nshards = "1"
	} else {
		nshards = strconv.Itoa(int(*hdb.Spec.Config.NShards))
	}

	configMap.Data = map[string]string{
		"NSHARDS": nshards,
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
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Info("Creating config map", "name", configMap.Name)

		if err = ctrl.SetControllerReference(hdb, configMap, r.Scheme); err != nil {
			return
		}
		return r.Create(ctx, configMap)
	} else if err != nil {
		return
	}

	if !equality.Semantic.DeepEqual(existing.Data, configMap.Data) {
		logger.Info("Updating config map")
		r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingConfigMap", "")

		existing.Data = configMap.Data
		err = r.Update(ctx, existing)
		if err != nil {
			return err
		}
	}
	return
}
