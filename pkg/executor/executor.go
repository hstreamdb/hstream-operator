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
	config    *rest.Config
	Clientset *kubernetes.Clientset
}

func NewRemoteExecutor(config *rest.Config) (*RemoteExecutor, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &RemoteExecutor{
		config:    config,
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

	exec, err := remotecommand.NewSPDYExecutor(e.config, http.MethodPost, req.URL())
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
