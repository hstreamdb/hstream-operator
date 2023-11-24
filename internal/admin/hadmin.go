package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	"github.com/hstreamdb/hstream-operator/pkg/executor"
	"github.com/hstreamdb/hstream-operator/pkg/selector"
	jsoniter "github.com/json-iterator/go"
	"k8s.io/client-go/rest"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type hAdminClient struct {
	hdb            *hapi.HStreamDB
	selector       *selector.Selector
	executor       *executor.PodExecutor
	legacyExecutor *remoteExecutor
	log            logr.Logger
}

// NewAdminClient generates an Admin client for a hStream
func NewHAdminClient(hdb *hapi.HStreamDB, restConfig *rest.Config, log logr.Logger) HAdminClient {
	legacyExecutor := NewExecutor(restConfig)
	e, _ := executor.NewPodExecutor(restConfig)
	selector := selector.NewSelector(e.Clientset)

	return &hAdminClient{
		hdb:            hdb,
		selector:       selector,
		executor:       e,
		legacyExecutor: legacyExecutor,
		log: log.WithValues("namespace", hdb.Namespace).
			WithValues("instance", hdb.Name),
	}
}

// BootstrapHServer init HServer nodes in HStreamDB cluster.
func (ac *hAdminClient) BootstrapHServer() error {
	command := executor.Command{
		Command: "hadmin",
		Args:    []string{"server", "init"},
	}

	pods, err := ac.selector.GetPods(ac.hdb.Namespace,
		&map[string]string{hapi.ComponentKey: string(hapi.ComponentTypeAdminServer)}, nil)

	_, err = ac.executor.RunCommandInPod(pods[0].Name, ac.hdb.Namespace, command)

	return err
}

// BootstrapHStore init hStore cluster
func (ac *hAdminClient) BootstrapHStore(metadataReplication int32) error {
	command := executor.Command{
		Command: "hadmin",
		Args: []string{"store", "nodes-config", "bootstrap",
			"--metadata-replicate-across", fmt.Sprintf("'node:%d'", metadataReplication)},
	}

	pods, err := ac.selector.GetPods(ac.hdb.Namespace,
		&map[string]string{hapi.ComponentKey: string(hapi.ComponentTypeAdminServer)}, nil)

	_, err = ac.executor.RunCommandInPod(pods[0].Name, ac.hdb.Namespace, command)

	return err
}

func (ac *hAdminClient) MaintenanceHStore(action MaintenanceAction, args []string) error {
	command := executor.Command{
		Command: "hadmin",
		Args:    append([]string{"store", "maintenance", string(action)}, args...),
	}

	pods, err := ac.selector.GetPods(ac.hdb.Namespace,
		&map[string]string{hapi.ComponentKey: string(hapi.ComponentTypeAdminServer)}, nil)

	_, err = ac.executor.RunCommandInPod(pods[0].Name, ac.hdb.Namespace, command)

	return err
}

func (ac *hAdminClient) GetHMetaStatus() (status HMetaStatus, err error) {
	hmetaAddr := ""
	namespace := ""
	if ac.hdb.Spec.ExternalHMeta != nil {
		namespace = ac.hdb.Spec.ExternalHMeta.Namespace
		hmetaAddr = ac.hdb.Spec.ExternalHMeta.Host + ":" + strconv.Itoa(int(ac.hdb.Spec.ExternalHMeta.Port))
	} else {
		namespace = ac.hdb.Namespace
		svc := internal.GetHeadlessService(ac.hdb, hapi.ComponentTypeHMeta)
		hmetaAddr = fmt.Sprintf("%s:%d", svc.Name, constants.DefaultHMetaPort.ContainerPort)
	}

	resp, statusCode, err := ac.legacyExecutor.GetAPIByService(namespace, hmetaAddr, "nodes")
	if err != nil {
		err = fmt.Errorf("get HMeta status failed. %w", err)
		return
	}
	if statusCode != http.StatusOK {
		err = fmt.Errorf("service unavailable: %s", jsoniter.Get(resp, "message").ToString())
		return
	}

	err = json.Unmarshal(resp, &status.Nodes)
	if err != nil {
		err = fmt.Errorf("unmarshal HMeta staus failed. %w", err)
		return
	}
	return
}

// func checkStoreInit(output string) (skipSubCmd bool, err error) {
// 	if strings.Contains(output, "Successfully bootstrapped the cluster") {
// 		return
// 	}

// 	err = fmt.Errorf("hstore init failed: %s", output)
// 	return
// }

// func checkServerInit(output string) (skipSubCmd bool, err error) {
// 	if strings.Contains(output, "Server successfully received init signal") {
// 		return
// 	}

// 	err = fmt.Errorf("hserver init failed: %s", output)
// 	return
// }
