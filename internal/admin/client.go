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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/pkg/executor"
	"github.com/hstreamdb/hstream-operator/pkg/selector"
	"k8s.io/client-go/rest"
)

type AdminClient struct {
	hdb      *v1alpha2.HStreamDB
	selector *selector.Selector
	executor *executor.RemoteExecutor
	log      logr.Logger
}

// NewAdminClient generates an admin client for communicating with the HStreamDB cluster.
func NewAdminClient(hdb *v1alpha2.HStreamDB, restConfig *rest.Config, log logr.Logger) IAdminClient {
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

func (ac *AdminClient) call(cmd string, args ...string) (string, error) {
	command := executor.Command{
		Command: cmd,
		Args:    args,
	}

	pods, err := ac.selector.GetPods(
		ac.hdb.Namespace,
		&map[string]string{v1alpha2.ComponentKey: string(v1alpha2.ComponentTypeAdminServer)},
		nil,
	)
	if err != nil {
		return "", err
	}

	return ac.executor.RunCommandInPod(pods[0].Name, ac.hdb.Namespace, command)
}

// CallServer call hadmin server command with args.
func (ac *AdminClient) CallServer(args ...string) (string, error) {
	return ac.call("hadmin", append([]string{"server"}, args...)...)
}

func (ac *AdminClient) CallKafkaServer(args ...string) (string, error) {
	return ac.call("hstream-kafka-cli", append([]string{"node"}, args...)...)
}

// CallStore call hadmin store command with args.
func (ac *AdminClient) CallStore(args ...string) (string, error) {
	return ac.call("hadmin", append([]string{"store"}, args...)...)
}

func (ac *AdminClient) MaintenanceStore(action MaintenanceAction, args ...string) (string, error) {
	return ac.call("hadmin", append([]string{"store", "maintenance", string(action)}, args...)...)
}

func (ac *AdminClient) GetInternalRqliteStatus() (status v1alpha2.InternalRqliteStatus, err error) {
	output, err := ac.executor.AccessServiceProxy(ac.hdb.Spec.Rqlite.Namespace, ac.hdb.Spec.Rqlite.Host+":"+fmt.Sprint(ac.hdb.Spec.Rqlite.Port), "nodes")
	if err != nil {
		err = fmt.Errorf("failed to get rqlite status: %w", err)

		return
	}

	err = json.Unmarshal(output, &status.Nodes)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal rqlite staus: %w", err)

		return
	}

	return
}
