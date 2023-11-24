package executor

import (
	"bytes"
	"fmt"
	"net/http"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type PodExecutor struct {
	config    *rest.Config
	Clientset *kubernetes.Clientset
}

func NewPodExecutor(config *rest.Config) (*PodExecutor, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &PodExecutor{
		config:    config,
		Clientset: clientset,
	}, nil
}

func (e *PodExecutor) RunCommandInPod(podName, namespace string, command Command) (string, error) {
	req := e.Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: command.getCommand(),
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
		return "", fmt.Errorf("an error occurred while executing the command: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
