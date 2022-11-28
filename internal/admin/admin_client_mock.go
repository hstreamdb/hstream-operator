package admin

import (
	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"k8s.io/client-go/rest"
)

type mockAdminClient struct {
	status HStreamStatus
}

func (ac *mockAdminClient) BootstrapHStore(ip string, port int) error {
	ac.status.HStoreInited = true
	return nil
}

func (ac *mockAdminClient) BootstrapHServer(ip string, port int) error {
	return nil
}

func (ac *mockAdminClient) GetStatus(ip string, port int) (HStreamStatus, error) {
	return ac.status, nil
}

type mockAdminClientProvider struct {
	client *mockAdminClient
}

func (m *mockAdminClientProvider) GetAdminClient(hdb *appsv1alpha1.HStreamDB) AdminClient {
	return m.client
}

// NewMockAdminClientProvider generates a client provider for talking to real hStream.
func NewMockAdminClientProvider(restConfig *rest.Config, log logr.Logger) AdminClientProvider {
	return &mockAdminClientProvider{
		client: &mockAdminClient{},
	}
}
