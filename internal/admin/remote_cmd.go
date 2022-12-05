package admin

import (
	"bytes"
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
	"time"
)

type Executor struct {
	clientSet *kubernetes.Clientset
	Config    rest.Config
}

func NewExecutor(config *rest.Config) *Executor {
	clientSet, _ := kubernetes.NewForConfig(config)
	return &Executor{
		clientSet: clientSet,
		Config:    *config,
	}
}

func (e *Executor) getPodNameByLabel(namespace string, label map[string]string) (name string, err error) {
	core := e.clientSet.CoreV1()
	pods, err := core.Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.FormatLabels(label),
		FieldSelector: fields.Set{"status.phase": "Running"}.String(),
		Limit:         1,
	})

	if err != nil {
		return
	}

	if len(pods.Items) == 0 {
		err = fmt.Errorf("couldn't find any pod with label %v", label)
		return
	}

	name = pods.Items[0].Name
	return
}

func (e *Executor) ExecToPodByLabel(namespace string, label map[string]string,
	containerName, command string, timeout time.Duration) (output string, err error) {

	podName, err := e.getPodNameByLabel(namespace, label)
	if err != nil {
		err = fmt.Errorf("couldn't get any running pod that has label %v", label)
		return
	}
	return e.ExecToPod(namespace, podName, containerName, command, timeout)
}

func (e *Executor) ExecToPod(namespace string, targetPod, containerName, command string, timeout time.Duration) (
	output string, err error) {

	req := e.clientSet.CoreV1().RESTClient().Post().Resource("pods").Name(targetPod).
		Namespace(namespace).SubResource("exec").Timeout(timeout)
	option := &v1.PodExecOptions{
		Command:   []string{"sh", "-c", command},
		Container: containerName,
		Stdout:    true,
		Stderr:    true,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(&e.Config, http.MethodPost, req.URL())
	if err != nil {
		err = fmt.Errorf("error while creating Executor: %v", err)
		return
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil || stderr.Len() != 0 {
		err = fmt.Errorf("error in Stream: %v, stderr: %s", err, stderr.String())
		return
	}

	output = stdout.String()
	return
}
