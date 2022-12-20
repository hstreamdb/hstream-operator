package internal

import (
	"errors"
	jsoniter "github.com/json-iterator/go"
	"sort"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	HStoreDataPath   = "/data/logdevice"
	HStoreConfigPath = "/etc/logdevice"
)

const (
	LogDeviceConfig string = "config-map"
	NShardsConfig   string = "nshards-map"
)

type ConfigMap struct {
	// pod.volumeMount.name
	MountName string
	// pod.volumeMount.mountPath
	MountPath string
	// spec.volume.name the full config map name is {hdb.name}-suffix
	MapNameSuffix string
	MapKey        string
	MapPath       string
}

type ConfigmapSet struct {
	cms map[string]*ConfigMap
}

var ConfigMaps = ConfigmapSet{
	cms: map[string]*ConfigMap{
		LogDeviceConfig: {
			MountName:     LogDeviceConfig,
			MountPath:     HStoreConfigPath,
			MapNameSuffix: "logdevice-config",
			MapKey:        "config.json",
			MapPath:       "config.json",
		},
		NShardsConfig: {
			MountName:     NShardsConfig,
			MountPath:     HStoreDataPath,
			MapNameSuffix: "nshards",
			MapKey:        "NSHARDS",
			MapPath:       "NSHARDS",
		},
	},
}

// Visit visits the config map in lexicographical order, calling fn for each.
func (c *ConfigmapSet) Visit(fn func(m ConfigMap)) {
	for _, flag := range sortConfigMaps(c.cms) {
		fn(*c.cms[flag])
	}
}

// Get returns the config map of given name
func (c *ConfigmapSet) Get(name string) (ConfigMap, bool) {
	if m, ok := c.cms[name]; ok {
		return *m, true
	}
	return ConfigMap{}, false
}

// sortConfigMaps returns the flags as a slice in lexicographical sorted order.
func sortConfigMaps(cms map[string]*ConfigMap) []string {
	result := make([]string, len(cms))
	i := 0
	for name := range cms {
		result[i] = name
		i++
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

var defaultLogDeviceConfig = map[string]any{
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
				"node": 1,
			},
		},
		"config_log_snapshots": map[string]any{
			"replicate_across": map[string]any{
				"node": 1,
			},
		},
		"event_log_deltas": map[string]any{
			"replicate_across": map[string]any{
				"node": 1,
			},
		},
		"event_log_snapshots": map[string]any{
			"replicate_across": map[string]any{
				"node": 1,
			},
		},
		"maintenance_log_deltas": map[string]any{
			"replicate_across": map[string]any{
				"node": 1,
			},
		},
		"maintenance_log_snapshots": map[string]any{
			"replicate_across": map[string]any{
				"node": 1,
			},
		},
	},
	"metadata_logs": map[string]any{
		"replicate_across": map[string]any{
			"node": 1,
		},
	},
	"rqlite": map[string]string{
		"rqlite_uri": "ip://rqlite-svc.default:4001",
	},
	"version": 1,
}

func GetLogDeviceConfig() map[string]any {
	return defaultLogDeviceConfig
}

func ParseLogDeviceConfig(raw []byte) (config map[string]any, err error) {
	config = make(map[string]any)
	if len(raw) != 0 {
		if !jsoniter.Valid(raw) {
			err = errors.New("incorrect json raw")
			return
		}
		_ = json.Unmarshal(raw, &config)
	}
	return
}
