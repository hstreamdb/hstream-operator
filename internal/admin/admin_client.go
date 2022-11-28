package admin

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
)

type AdminClient interface {
	BootstrapHStore(ip string, port int) error
	BootstrapHServer(ip string, port int) error
	GetStatus(ip string, port int) (HStreamStatus, error)
}

type HStreamStatus struct {
	HStoreInited bool
	HServer      string
	HStore       string
}

// AdminClientProvider provides an abstraction for creating clients that
// communicate with the hstreamdb.
type AdminClientProvider interface {
	// GetAdminClient generates a client for performing administrative actions
	// against the hstreamdb.
	GetAdminClient(hdb *appsv1alpha1.HStreamDB) AdminClient
}
