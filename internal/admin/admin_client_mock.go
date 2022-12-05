package admin

import (
	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"k8s.io/client-go/rest"
)

type mockAdminClient struct {
}

func (ac *mockAdminClient) BootstrapHStore() error {

	return nil
}

func (ac *mockAdminClient) BootstrapHServer() error {
	return nil
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
