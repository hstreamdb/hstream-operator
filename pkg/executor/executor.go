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

package executor

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type RemoteExecutor struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
}

func NewRemoteExecutor(config *rest.Config) (*RemoteExecutor, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &RemoteExecutor{
		Config:    config,
		Clientset: clientset,
	}, nil
}

func (e *RemoteExecutor) RunCommandInPod(podName, namespace string, command Command) (string, error) {
	req := e.Clientset.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec")
	option := &v1.PodExecOptions{
		Command: command.GetCommand(),
		Stdout:  true,
		Stderr:  true,
	}

	req.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(e.Config, http.MethodPost, req.URL())
	if err != nil {
		return "", fmt.Errorf("an error occurred while creating the executor: %w", err)
	}

	var stdout, stderr bytes.Buffer

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil || stderr.Len() != 0 {
		return "", fmt.Errorf("an error occurred while executing the command: %w, command: %s, stderr: %s", err, command.ToString(), stderr.String())
	}

	return stdout.String(), nil
}

func (e *RemoteExecutor) AccessServiceProxy(namespace, serviceName, path string) (output []byte, err error) {
	return e.Clientset.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Resource("services").
		Name(serviceName).
		SubResource("proxy").
		Suffix(path).DoRaw(context.TODO())
}
