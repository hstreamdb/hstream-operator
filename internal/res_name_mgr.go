package internal

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"strings"
)

func GetResNameOnPanic(hdb *hapi.HStreamDB, shortName string) string {
	if shortName == "" {
		panic("short name is empty")
	}
	return GetResNameWithDefault(hdb, shortName, "")
}

// GetResNameWithDefault get resource name with short name, it will use default name if short name is empty
func GetResNameWithDefault(hdb *hapi.HStreamDB, shortName, defaultName string) string {
	buf := strings.Builder{}
	buf.WriteString(hdb.Name)
	buf.WriteRune('-')
	if shortName == "" {
		buf.WriteString(defaultName)
	} else {
		buf.WriteString(shortName)
	}
	return buf.String()
}
