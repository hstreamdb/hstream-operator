package admin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// defaultCliTimeout is the default timeout for running commands.
var defaultCliTimeout = 15 * time.Second

// remoteCommand describes a command to be run in a container.
type remoteCommand struct {
	// hdbComponentType is the component we need to run commands in.
	hdbComponentType hapi.ComponentType

	containerName string

	binary string

	args []string

	// timeout provides a way to overwrite the default timeout.
	timeout time.Duration
}

func (rc remoteCommand) getCommand() string {
	return fmt.Sprintf("%s %s", rc.binary, strings.Join(rc.args, " "))
}

func (rc remoteCommand) getTimeout() time.Duration {
	if rc.timeout != 0 {
		return rc.timeout
	}

	return defaultCliTimeout
}

type remoteExecutor struct {
	clientSet  *kubernetes.Clientset
	httpClient *http.Client
	Config     rest.Config
}

func NewExecutor(config *rest.Config) *remoteExecutor {
	config.Timeout = 10 * time.Second
	httpClient, _ := rest.HTTPClientFor(config)
	clientSet, _ := kubernetes.NewForConfig(config)

	return &remoteExecutor{
		clientSet:  clientSet,
		httpClient: httpClient,
		Config:     *config,
	}
}

func (e *remoteExecutor) ExecToPodByLabel(namespace string, label map[string]string,
	containerName, command string, timeout time.Duration) (output string, err error) {

	podName, err := e.getPodNameByLabel(namespace, label)
	if err != nil {
		err = fmt.Errorf("couldn't get any running pod that has label %v", label)
		return
	}

	return e.ExecToPod(namespace, podName, containerName, command, timeout)
}

func (e *remoteExecutor) getPodNameByLabel(namespace string, label map[string]string) (name string, err error) {
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

func (e *remoteExecutor) ExecToPod(namespace string, targetPod, containerName, command string, timeout time.Duration) (
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

// GetAPIByService call pod http api by service
// the supported formats for the serviceName segment of the URL are:
// <service_name> - proxies to the default or unnamed port using http
// <service_name>:<port_name> - proxies to the specified port name or port number using http
// https:<service_name>: - proxies to the default or unnamed port using https (note the trailing colon)
// https:<service_name>:<port_name> - proxies to the specified port name or port number using https
// More info: https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster-services/#manually-constructing-apiserver-proxy-urls
func (e *remoteExecutor) GetAPIByService(namespace, serviceName, path string) (output []byte, statusCode int, err error) {
	req := e.clientSet.CoreV1().RESTClient().Get().Resource("services").
		Namespace(namespace).SubResource("proxy").
		Name(serviceName).Suffix(path)

	resp, err := e.httpClient.Get(req.URL().String())
	if err != nil {
		err = fmt.Errorf("err from http request %w", err)
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	statusCode = resp.StatusCode
	output, err = io.ReadAll(resp.Body)
	return
}
