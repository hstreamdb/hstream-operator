package admin

import (
	"github.com/go-logr/logr"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MaintenanceAction string

const (
	MaintenanceActionApply MaintenanceAction = "apply"
)

type HAdminClient interface {
	BootstrapHServer() error
	BootstrapHStore(metadataReplication int32) error
	MaintenanceHStore(action MaintenanceAction, args []string) error
	GetHMetaStatus() (HMetaStatus, error)
}

type HMetaClient interface {
	GetHMetaStatus() (HMetaStatus, error)
}

// AdminClientProvider provides an abstraction for creating clients that
// communicate with the HStreamDB cluster.
type AdminClientProvider interface {
	GetHAdminClient(hdb *hapi.HStreamDB) HAdminClient
	// GetHMetaClient(hdb *hapi.HStreamDB) HMetaClient
}

type adminClientProvider struct {
	// client defines the ctrl client.
	client client.Client

	// restConfig defines k8s client config.
	restConfig *rest.Config

	// log defines the logger for the admin client.
	log logr.Logger
}

func (p *adminClientProvider) GetHAdminClient(hdb *hapi.HStreamDB) HAdminClient {
	return NewHAdminClient(hdb, p.restConfig, p.log)
}

func NewAdminClientProvider(restConfig *rest.Config, log logr.Logger) AdminClientProvider {
	return &adminClientProvider{
		restConfig: restConfig,
		log:        log.WithName("Admin Client"),
	}
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
