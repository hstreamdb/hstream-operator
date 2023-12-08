package admin

import (
	"fmt"

	"github.com/go-logr/logr"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"k8s.io/client-go/rest"
)

type mockAdminClient struct {
	hdb *hapi.HStreamDB
}

func (ac *mockAdminClient) CallServer(args ...string) (string, error) {
	panic("unimplemented")
}

func (ac *mockAdminClient) CallStore(args ...string) (string, error) {
	panic("unimplemented")
}

// MaintenanceHStore implements HAdminClient.

func (*mockAdminClient) MaintenanceStore(action MaintenanceAction, args ...string) (string, error) {
	panic("unimplemented")
}

func (ac *mockAdminClient) GetHMetaStatus() (status HMetaStatus, err error) {
	for i := 0; i < int(ac.hdb.Spec.HMeta.Replicas); i++ {
		status.Nodes[fmt.Sprint("nodeId-", i)] = HMetaNode{
			Reachable: true,
			Leader:    false,
			Error:     "",
		}
	}
	return
}

type mockAdminClientProvider struct {
	client *mockAdminClient
}

func (m *mockAdminClientProvider) GetAdminClient(hdb *hapi.HStreamDB) IAdminClient {
	m.client.hdb = hdb
	return m.client
}

// NewMockAdminClientProvider generates a client provider for talking to real hStream.
func NewMockAdminClientProvider(*rest.Config, logr.Logger) AdminClientProvider {
	return &mockAdminClientProvider{
		client: &mockAdminClient{},
	}
}
