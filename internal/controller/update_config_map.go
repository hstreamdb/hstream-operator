package controller

import (
	"context"
	"fmt"
	"strconv"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	jsoniter "github.com/json-iterator/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	maxRecommendedLogReplication = 3
)

type updateConfigMap struct {
}

func (u updateConfigMap) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	configMap, err := getLogDeviceConfigMap(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	if err = u.createOrUpdate(ctx, r, hdb, &configMap); err != nil {
		return &requeue{curError: err}
	}

	nShardsMap, err := getNShardsMap(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	// nshard doesn't support update
	if _, err = u.create(ctx, r, hdb, &nShardsMap); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func getLogDeviceConfigMap(hdb *hapi.HStreamDB) (configMap corev1.ConfigMap, err error) {
	config, err := getLogDeviceConfig(hdb)
	if err != nil {
		return
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

func getNShardsMap(hdb *hapi.HStreamDB) (configMap corev1.ConfigMap, err error) {
	cm, has := internal.ConfigMaps.Get(internal.NShardsConfig)
	if !has {
		err = fmt.Errorf("no such config map %s", internal.NShardsConfig)
		return
	}

	nShards := getMinNShards(hdb)
	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      internal.GetResNameOnPanic(hdb, "nshards"),
			Namespace: hdb.GetNamespace(),
		},
		Data: map[string]string{
			cm.MapKey: strconv.Itoa(int(nShards)),
		},
	}
	return
}

func (u updateConfigMap) createOrUpdate(ctx context.Context, r *HStreamDBReconciler,
	hdb *hapi.HStreamDB, configMap *corev1.ConfigMap) (err error) {

	logger := log.WithValues("namespace", configMap.Namespace,
		"instance", hdb.Name, "name", configMap.Name, "reconciler", "UpdateConfigMap")

	existing, err := u.create(ctx, r, hdb, configMap)
	if err != nil || existing == nil {
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

func (u updateConfigMap) create(ctx context.Context, r *HStreamDBReconciler,
	hdb *hapi.HStreamDB, newConfigMap *corev1.ConfigMap) (existing *corev1.ConfigMap, err error) {

	logger := log.WithValues("namespace", newConfigMap.Namespace,
		"instance", hdb.Name, "name", newConfigMap.Name, "reconciler", "UpdateConfigMap")

	existing = &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKeyFromObject(newConfigMap), existing)
	if err != nil {
		existing = nil
		if k8sErrors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(hdb, newConfigMap, r.Scheme); err == nil {
				logger.Info("Creating config map", "name", newConfigMap.Name)
				err = r.Create(ctx, newConfigMap)
				return
			}
		}
		return
	}
	return
}

func getLogDeviceConfig(hdb *hapi.HStreamDB) (config map[string]any, err error) {
	// parse json config from cr
	config = make(map[string]any)
	raw := hdb.Spec.Config.LogDeviceConfig.Raw
	if len(raw) != 0 {
		if !jsoniter.Valid(raw) {
			err = fmt.Errorf("parse log device config failed: invalid json format")
			return
		}
		_ = json.Unmarshal(raw, &config)
	}

	// append HMeta addr to the logDevice config
	hmetaAddr, err := getHMetaAddr(hdb)
	if err != nil {
		return
	}

	// merge default config if user doesn't set
	defaultLogDeviceConfig := generateDefaultConfig(hdb.Spec.HStore.Replicas, hmetaAddr)
	for key, value := range defaultLogDeviceConfig {
		if _, ok := config[key]; !ok {
			config[key] = value
		}
	}
	return
}

func generateDefaultConfig(podReplicas int32, hmetaAddr string) map[string]any {
	replicaAcross := getRecommendedLogReplicaAcross(podReplicas)

	return map[string]any{
		"server_settings": map[string]any{
			"enable-nodes-configuration-manager":                  "true",
			"use-nodes-configuration-manager-nodes-configuration": "true",
			"enable-node-self-registration":                       "true",
			"enable-cluster-maintenance-state-machine":            "true",
		},
		"client_settings": map[string]any{
			"enable-nodes-configuration-manager":                  "true",
			"use-nodes-configuration-manager-nodes-configuration": "true",
			"admin-client-capabilities":                           "true",
		},
		"cluster": "hstore",
		"internal_logs": map[string]any{
			"config_log_deltas": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
			"config_log_snapshots": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
			"event_log_deltas": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
			"event_log_snapshots": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
			"maintenance_log_deltas": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
			"maintenance_log_snapshots": map[string]any{
				"replicate_across": map[string]any{
					"node": replicaAcross,
				},
			},
		},
		"rqlite": map[string]string{
			"rqlite_uri": "ip://" + hmetaAddr,
		},
		"version": 1,
	}
}

// the recommended max replica across is 3 and must be less than or equal to pod num
func getRecommendedLogReplicaAcross(podReplicas int32) int32 {
	if podReplicas <= maxRecommendedLogReplication {
		return podReplicas
	}
	return maxRecommendedLogReplication
}

func getMinNShards(hdb *hapi.HStreamDB) int32 {
	if hdb.Spec.Config.NShards == 0 {
		return 1
	}
	return hdb.Spec.Config.NShards
}
