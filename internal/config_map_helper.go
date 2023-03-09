package internal

import (
	"sort"
)

const (
	HStoreDataPath   = "/data/logdevice"
	HStoreConfigPath = "/etc/logdevice"
	HMetaDataPath    = "/rqlite/file"
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
