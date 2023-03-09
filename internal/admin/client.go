package admin

import (
	"fmt"
	"github.com/go-logr/logr"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	jsoniter "github.com/json-iterator/go"
	"k8s.io/client-go/rest"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// defaultCLITimeout is the default timeout for CLI commands.
var defaultCLITimeout = 15 * time.Second

type realAdminClientProvider struct {
	// log implementation for logging output
	log logr.Logger
	// restConfig k8s cluster restapi config
	restConfig *rest.Config
}

// GetAdminClient generates a client for performing administrative actions
// against the hStream.
func (p *realAdminClientProvider) GetAdminClient(hdb *hapi.HStreamDB) AdminClient {
	return NewAdminClient(hdb, p.restConfig, p.log)
}

// NewAdminClientProvider generates a client provider for talking to real hStream.
func NewAdminClientProvider(restConfig *rest.Config, log logr.Logger) AdminClientProvider {
	return &realAdminClientProvider{
		log:        log.WithName("adminClient"),
		restConfig: restConfig,
	}
}

type adminClient struct {
	hdb        *hapi.HStreamDB
	remoteExec *Executor
	log        logr.Logger
}

// NewAdminClient generates an Admin client for a hStream
func NewAdminClient(hdb *hapi.HStreamDB, restConfig *rest.Config, log logr.Logger) AdminClient {
	return &adminClient{
		hdb:        hdb,
		remoteExec: NewExecutor(restConfig),
		log: log.WithValues("namespace", hdb.Namespace).
			WithValues("instance", hdb.Name),
	}
}

type cmdOrder struct {
	args        []string
	resultCheck func(output string) (skipSubCmd bool, err error)
	subCmd      *cmdOrder
}

// BootstrapHStore init hStore cluster
func (ac *adminClient) BootstrapHStore(metadataReplication int32) error {
	command := cliCommand{
		// run the bootstrap cmd in the admin server pod
		targetComponent: string(hapi.ComponentTypeAdminServer),
		containerName:   ac.hdb.Spec.AdminServer.Container.Name,
		args: []string{"store", "nodes-config", "bootstrap",
			"--metadata-replicate-across", fmt.Sprintf("'node:%d'", metadataReplication)},
	}

	flags := internal.FlagSet{}
	if err := flags.Parse(ac.hdb.Spec.AdminServer.Container.Args); err != nil {
		return fmt.Errorf("parse admin server args failed. %w", err)
	}
	args := flags.Flags()
	if port, ok := args["admin-port"]; ok {
		command.args = append(command.args, "--port", port)
	}

	output, err := ac.runCommandInPod(command)
	if err != nil {
		return err
	}

	_, err = checkStoreInit(output)
	return err
}

// BootstrapHServer init hServer cluster
func (ac *adminClient) BootstrapHServer() error {
	command := cliCommand{
		targetComponent: string(hapi.ComponentTypeHServer),
		containerName:   ac.hdb.Spec.HServer.Container.Name,
		args:            []string{"server", "init"},
	}

	flags := internal.FlagSet{}
	if err := flags.Parse(ac.hdb.Spec.HServer.Container.Args); err != nil {
		return fmt.Errorf("parse hServer args failed. %w", err)
	}
	args := flags.Flags()
	if port, ok := args["port"]; ok {
		command.args = append(command.args, "--port", port)
	}

	output, err := ac.runCommandInPod(command)
	if err != nil {
		return err
	}

	_, err = checkServerInit(output)
	return err
}

func (ac *adminClient) GetHMetaStatus() (status HMetaStatus, err error) {
	hmetaAddr := ""
	namespace := ""
	if ac.hdb.Spec.ExternalHMeta != nil {
		namespace = ac.hdb.Spec.ExternalHMeta.Namespace
		hmetaAddr = ac.hdb.Spec.ExternalHMeta.Host + ":" + strconv.Itoa(int(ac.hdb.Spec.ExternalHMeta.Port))
	} else {
		namespace = ac.hdb.Namespace
		svc := internal.GetService(ac.hdb, hapi.ComponentTypeHMeta)
		hmetaAddr = fmt.Sprintf("%s:port", svc.Name)
	}

	resp, statusCode, err := ac.remoteExec.GetAPIByService(namespace, hmetaAddr, "nodes")
	if err != nil {
		err = fmt.Errorf("get hmeta status failed. %w", err)
		return
	}
	if statusCode != http.StatusOK {
		err = fmt.Errorf("service unavailable: %s", jsoniter.Get(resp, "message").ToString())
		return
	}

	err = json.Unmarshal(resp, &status.Nodes)
	if err != nil {
		err = fmt.Errorf("unmarshal hmeta staus failed. %w", err)
		return
	}
	return
}

func checkStoreInit(output string) (skipSubCmd bool, err error) {
	if strings.Contains(output, "Successfully bootstrapped the cluster") {
		return
	}

	err = fmt.Errorf("hstore init failed: %s", output)
	return
}

func checkServerInit(output string) (skipSubCmd bool, err error) {
	if strings.Contains(output, "Server successfully received init signal") {
		return
	}

	err = fmt.Errorf("hserver init failed: %s", output)
	return
}

func (ac *adminClient) runCommandInPod(cmd cliCommand) (string, error) {
	if cmd.binary == "" {
		cmd.binary = "hadmin"
	}

	containerName := cmd.containerName
	// use component type as default container name if user doesn't define name
	if containerName == "" {
		containerName = cmd.targetComponent
	}

	remoteCmdToExec := fmt.Sprintf("%s %s", cmd.binary, strings.Join(cmd.args, " "))

	targetPodLabel := map[string]string{hapi.ComponentKey: cmd.targetComponent}
	output, err := ac.remoteExec.ExecToPodByLabel(ac.hdb.Namespace, targetPodLabel,
		containerName, remoteCmdToExec, cmd.getTimeout())
	if err != nil {
		//ac.log.Error(err, "Error from "+cmd.binary+" command", "command", remoteCmdToExec)
		return "", err
	}
	return output, nil
}

// cliCommand describes a command that we are running against hstream.
type cliCommand struct {
	// binary is the binary to run.
	binary string

	// the component will run the cmd
	targetComponent string
	// the container in the target component pod
	containerName string

	args []string

	// timeout provides a way to overwrite the default cli timeout.
	timeout time.Duration
}

// getTimeout returns the timeout for the command
func (command cliCommand) getTimeout() time.Duration {
	if command.timeout != 0 {
		return command.timeout
	}
	return defaultCLITimeout
}
