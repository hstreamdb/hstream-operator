/*
Copyright 2023 HStream Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admin

import (
	"fmt"
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

type AdminClient struct {
	hdb      *hapi.HStreamDB
	selector *selector.Selector
	executor *executor.RemoteExecutor
	log      logr.Logger
}

// NewAdminClient generates an Admin client for a hStream
func NewAdminClient(hdb *hapi.HStreamDB, restConfig *rest.Config, log logr.Logger) IAdminClient {
	e, _ := executor.NewRemoteExecutor(restConfig)
	selector := selector.NewSelector(e.Clientset)

	return &AdminClient{
		hdb:      hdb,
		selector: selector,
		executor: e,
		log: log.WithValues("namespace", hdb.Namespace).
			WithValues("instance", hdb.Name),
	}
}

func (ac *AdminClient) call(args ...string) (string, error) {
	command := executor.Command{
		Command: "hadmin",
		Args:    args,
	}

	pods, err := ac.selector.GetPods(
		ac.hdb.Namespace,
		&map[string]string{hapi.ComponentKey: string(hapi.ComponentTypeAdminServer)},
		nil,
	)
	if err != nil {
		return "", err
	}

	return ac.executor.RunCommandInPod(pods[0].Name, ac.hdb.Namespace, command)
}

// CallServer call hadmin server command with args.
func (ac *AdminClient) CallServer(args ...string) (string, error) {
	return ac.call(append([]string{"server"}, args...)...)
}

// CallStore call hadmin store command with args.
func (ac *AdminClient) CallStore(args ...string) (string, error) {
	return ac.call(append([]string{"store"}, args...)...)
}

func (ac *AdminClient) MaintenanceStore(action MaintenanceAction, args ...string) (string, error) {
	return ac.call(append([]string{"store", "maintenance", string(action)}, args...)...)
}

func (ac *AdminClient) GetHMetaStatus() (status HMetaStatus, err error) {
	namespace := ""
	hmetaAddr := ""

	if ac.hdb.Spec.ExternalHMeta != nil {
		namespace = ac.hdb.Spec.ExternalHMeta.Namespace
		hmetaAddr = ac.hdb.Spec.ExternalHMeta.Host + ":" + strconv.Itoa(int(ac.hdb.Spec.ExternalHMeta.Port))
	} else {
		namespace = ac.hdb.Namespace
		svc := internal.GetHeadlessService(ac.hdb, hapi.ComponentTypeHMeta)
		hmetaAddr = fmt.Sprintf("%s:%d", svc.Name, constants.DefaultHMetaPort.ContainerPort)
	}

	output, err := ac.executor.AccessServiceProxy(namespace, hmetaAddr, "nodes")
	if err != nil {
		err = fmt.Errorf("failed to get HMeta status: %w", err)

		return
	}

	err = json.Unmarshal(output, &status.Nodes)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal HMeta staus: %w", err)

		return
	}

	return
}
