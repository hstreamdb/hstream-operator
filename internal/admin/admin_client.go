package admin

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
)

type AdminClient interface {
	BootstrapHStore() error
	BootstrapHServer() error
}

// AdminClientProvider provides an abstraction for creating clients that
// communicate with the HStreamDB.
type AdminClientProvider interface {
	// GetAdminClient generates a client for performing administrative actions
	// against the hstreamdb.
	GetAdminClient(hdb *appsv1alpha1.HStreamDB) AdminClient
}
