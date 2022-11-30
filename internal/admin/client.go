package admin

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"k8s.io/client-go/rest"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// defaultCLITimeout is the default timeout for CLI commands.
var defaultCLITimeout = 10

type realAdminClientProvider struct {
	// log implementation for logging output
	log logr.Logger
	// restConfig k8s cluster restapi config
	restConfig *rest.Config
}

// GetAdminClient generates a client for performing administrative actions
// against the hStream.
func (p *realAdminClientProvider) GetAdminClient(hdb *appsv1alpha1.HStreamDB) AdminClient {
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
	hdb        *appsv1alpha1.HStreamDB
	remoteExec *Executor
	log        logr.Logger
}

// NewAdminClient generates an Admin client for a hStream
func NewAdminClient(hdb *appsv1alpha1.HStreamDB, restConfig *rest.Config, log logr.Logger) AdminClient {
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

func (ac *adminClient) GetStatus(ip string, port int) (s HStreamStatus, err error) {
	sPort := strconv.Itoa(port)
	output, _ := ac.runCommandInPod(cliCommand{
		args: []string{"store", "--host", ip, "--port", sPort, "status"},
	})

	ok, _ := checkStoreStatus(output)
	if ok {
		s.HStoreInited = true
	}

	// TODO: get all hStream status from http service
	return s, nil

}

// BootstrapHStore init hStore cluster
func (ac *adminClient) BootstrapHStore(ip string, port int) error {
	sPort := strconv.Itoa(port)
	// ignore err and check store status
	output, _ := ac.runCommandInPod(cliCommand{
		args: []string{"store", "--host", ip, "--port", sPort, "status"},
	})
	skipSubCmd, err := checkStoreStatus(output)
	if err != nil {
		return err
	}
	if skipSubCmd {
		return nil
	}

	output, err = ac.runCommandInPod(cliCommand{
		args: []string{"store", "--host", ip, "--port", sPort, "nodes-config", "bootstrap",
			"--metadata-replicate-across", fmt.Sprintf("'node:%d'", ac.hdb.Spec.Config.MetadataReplicateAcross)},
	})
	if err != nil {
		return err
	}
	_, err = checkStoreInit(output)
	return err
}

// BootstrapHServer init hserver cluster
func (ac *adminClient) BootstrapHServer(ip string, port int) error {
	sPort := strconv.Itoa(port)
	// ignore err and check store status
	output, _ := ac.runCommandInPod(cliCommand{
		args: []string{"server", "--host", ip, "--port", sPort, "status"},
	})

	skipSubCmd, err := checkServerStatus(output)
	if err != nil {
		return err
	}
	if skipSubCmd {
		return nil
	}

	output, err = ac.runCommandInPod(cliCommand{
		args: []string{"server", "--host", ip, "--port", sPort, "init"},
	})
	if err != nil {
		return err
	}
	_, err = checkServerInit(output)
	return err
}

func checkStoreStatus(output string) (skipSubCmd bool, err error) {
	if strings.Contains(output, "ALIVE") {
		// return true for skipping sub cmd
		return true, nil
	}
	// TODO: check error
	return false, nil
}

func checkStoreInit(output string) (skipSubCmd bool, err error) {
	if strings.Contains(output, "Successfully bootstrapped the cluster") {
		return
	}

	err = fmt.Errorf("hstore init failed: %s", output)
	return
}

func checkServerStatus(output string) (skipSubCmd bool, err error) {
	if strings.Contains(output, "Running") {
		return true, nil
	}
	// TODO: check error
	return false, nil
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

	remoteCmdToExec := fmt.Sprintf("%s %s", cmd.binary, strings.Join(cmd.args, " "))

	// run the bootstrap cmd in the admin server pod
	podType := string(appsv1alpha1.ComponentTypeAdminServer)
	containerName := ac.hdb.Spec.AdminServer.Container.Name
	if containerName == "" {
		containerName = podType
	}
	label := map[string]string{appsv1alpha1.ComponentKey: podType}

	output, err := ac.remoteExec.ExecToPodByLabel(ac.hdb.Namespace, label, containerName, remoteCmdToExec, nil)
	if err != nil {
		ac.log.Error(err, "Error from hadmin command", "stdout", output, "command", remoteCmdToExec)
		return "", err
	}
	return output, nil
}

func (ac *adminClient) runCommand(cmd cliCommand) (string, error) {
	if cmd.binary == "" {
		cmd.binary = "hadmin"
	}

	args := strings.Join(cmd.args, " ")

	hardTimeout := cmd.getTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(hardTimeout))
	defer cancel()

	execCommand := exec.CommandContext(ctx, cmd.binary, args)
	ac.log.Info("Running command", "path", execCommand.Path, "args", execCommand.Args)

	output, err := execCommand.CombinedOutput()
	if err != nil {
		exitError, canCast := err.(*exec.ExitError)
		if canCast {
			ac.log.Error(exitError, "Error from hadmin command", "code",
				exitError.ProcessState.ExitCode(), "stdout", string(output), "stderr", string(exitError.Stderr))
		}
		return "", err
	}
	return string(output), nil
}

// cliCommand describes a command that we are running against hstream.
type cliCommand struct {
	// binary is the binary to run.
	binary string

	args []string

	// timeout provides a way to overwrite the default cli timeout.
	timeout time.Duration
}

// getTimeout returns the timeout for the command
func (command cliCommand) getTimeout() int {
	if command.timeout != 0 {
		return int(command.timeout.Seconds())
	}
	return defaultCLITimeout
}
