package admin

import (
	"github.com/go-logr/logr"
	"github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"k8s.io/client-go/rest"
)

type mockAdminClient struct {
	hdb *v1alpha2.HStreamDB
}

func (ac *mockAdminClient) CallServer(args ...string) (string, error) {
	panic("unimplemented")
}

func (ac *mockAdminClient) CallKafkaServer(args ...string) (string, error) {
	panic("unimplemented")
}

func (ac *mockAdminClient) CallStore(args ...string) (string, error) {
	panic("unimplemented")
}

// MaintenanceHStore implements HAdminClient.

func (*mockAdminClient) MaintenanceStore(action MaintenanceAction, args ...string) (string, error) {
	panic("unimplemented")
}

func (ac *mockAdminClient) GetInternalRqliteStatus() (status v1alpha2.InternalRqliteStatus, err error) {
	status.Nodes = map[string]v1alpha2.RqliteNode{
		"hstreamdb-internal-rqlite-0": {
			Id:        "hstreamdb-internal-rqlite-0",
			Addr:      "hstreamdb-internal-rqlite-0.hstreamdb-internal-rqlite-headless.hstreamdb.svc.cluster.local:4002",
			Reachable: true,
			Leader:    false,
		},
	}

	return
}

type mockAdminClientProvider struct {
	client *mockAdminClient
}

func (m *mockAdminClientProvider) GetAdminClient(hdb *v1alpha2.HStreamDB) IAdminClient {
	m.client.hdb = hdb

	return m.client
}

// NewMockAdminClientProvider generates a client provider for talking to real hStream.
func NewMockAdminClientProvider(*rest.Config, logr.Logger) AdminClientProvider {
	return &mockAdminClientProvider{
		client: &mockAdminClient{},
	}
}
