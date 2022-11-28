package admin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type Executor struct {
	Clientset kubernetes.Clientset
	Config    rest.Config
}

func NewExecutor(config *rest.Config) *Executor {
	clientset, _ := kubernetes.NewForConfig(config)
	return &Executor{
		Clientset: *clientset,
		Config:    *config,
	}
}

func (e *Executor) ExecToPodByLabel(namespace string, label map[string]string,
	containerName, command string, stdin io.Reader) (output string, err error) {

	core := e.Clientset.CoreV1()
	pods, err := core.Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.FormatLabels(label),
		Limit:         1,
	})
	if err != nil {
		return
	}

	if len(pods.Items) == 0 {
		err = fmt.Errorf("couldn't find any pod with label %v", label)
		return
	}

	pod := pods.Items[0]
	return e.ExecToPod(namespace, pod.Name, containerName, command, stdin)
}

func (e *Executor) ExecToPod(namespace string, podName, containerName, command string, stdin io.Reader) (
	output string, err error) {

	req := e.Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command:   []string{"sh", "-c", command},
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(&e.Config, "POST", req.URL())
	if err != nil {
		err = fmt.Errorf("error while creating Executor: %v", err)
		return
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		err = fmt.Errorf("error in Stream: %v", err)
		return
	}
	if stderr.Len() != 0 {
		err = errors.New(stderr.String())
		return
	}

	output = stdout.String()
	return
}
