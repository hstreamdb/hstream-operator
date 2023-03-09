package admin

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
)

type AdminClient interface {
	BootstrapHStore(metadataReplication int32) error
	BootstrapHServer() error
	GetHMetaStatus() (HMetaStatus, error)
}

// AdminClientProvider provides an abstraction for creating clients that
// communicate with the HStreamDB.
type AdminClientProvider interface {
	// GetAdminClient generates a client for performing administrative actions
	// against the hstreamdb.
	GetAdminClient(hdb *hapi.HStreamDB) AdminClient
}

type HMetaStatus struct {
	Nodes map[string]HMetaNode
}

type HMetaNode struct {
	Reachable bool
	Leader    bool
	Error     string
}

func (rs *HMetaStatus) IsAllReady() bool {
	if len(rs.Nodes) == 0 {
		return false
	}
	for _, node := range rs.Nodes {
		if !node.Reachable {
			return false
		}
	}
	return true
}
