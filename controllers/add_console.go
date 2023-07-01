package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	consoleEnvPortName       = "SERVER_PORT"
	consoleEnvHServerAddr    = "HSTREAM_PRIVATE_ADDRESS"
	consoleContainerPortName = "server-port"
)

const (
	consoleDefaultPort = 5177
)

var consoleEnvVars = []corev1.EnvVar{
	{
		Name:  consoleEnvPortName,
		Value: strconv.Itoa(consoleDefaultPort),
	},
}

var consolePort = corev1.ContainerPort{
	Name:          consoleContainerPortName,
	ContainerPort: consoleDefaultPort,
	Protocol:      corev1.ProtocolTCP,
}

type addConsole struct{}

func (a addConsole) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	if hdb.Spec.Console == nil {
		return nil
	}

	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "add console")

	deploy, err := a.getDeployment(hdb)
	if err != nil {
		return &requeue{curError: err}
	}

	existingDeploy := &appsv1.Deployment{}
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(&deploy), existingDeploy)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return &requeue{curError: err}
		}
		if err = ctrl.SetControllerReference(hdb, &deploy, r.Scheme); err != nil {
			return &requeue{curError: err}
		}

		logger.Info("Create console")
		if err = r.Client.Create(ctx, &deploy); err != nil {
			return &requeue{curError: err}
		}
		return nil
	}
	if !isHashChanged(&existingDeploy.ObjectMeta, &deploy.ObjectMeta) {
		return nil
	}

	logger.Info("Update console")
	r.Recorder.Event(hdb, corev1.EventTypeNormal, "UpdatingConsole", "")

	existingDeploy.Annotations = deploy.Annotations
	existingDeploy.Labels = deploy.Labels
	existingDeploy.Spec = deploy.Spec
	if err = r.Update(ctx, existingDeploy); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addConsole) getDeployment(hdb *hapi.HStreamDB) (deploy appsv1.Deployment, err error) {
	podTemplate, err := a.getPodTemplate(hdb)
	if err != nil {
		return
	}

	deploy = internal.GetDeployment(hdb, hdb.Spec.Console,
		&podTemplate, hapi.ComponentTypeConsole)
	return
}

func (a addConsole) getPodTemplate(hdb *hapi.HStreamDB) (spec corev1.PodTemplateSpec, err error) {
	console := hdb.Spec.Console

	containers, err := a.getContainer(hdb)
	if err != nil {
		return
	}

	spec = corev1.PodTemplateSpec{
		ObjectMeta: internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeConsole),
		Spec: corev1.PodSpec{
			Affinity:        console.Affinity,
			Tolerations:     console.Tolerations,
			NodeName:        console.NodeName,
			NodeSelector:    console.NodeSelector,
			SecurityContext: console.PodSecurityContext,
			InitContainers:  console.InitContainers,
			Containers:      containers,
			Volumes:         console.Volumes,
		},
	}

	spec.Name = hapi.ComponentTypeConsole.GetResName(hdb.Name)
	return
}

func (a addConsole) getContainer(hdb *hapi.HStreamDB) ([]corev1.Container, error) {
	console := hdb.Spec.Console
	container := corev1.Container{
		Image:           hdb.Spec.Console.Image,
		ImagePullPolicy: hdb.Spec.Console.ImagePullPolicy,
	}

	structAssign(&container, &console.Container)

	if container.Name == "" {
		container.Name = string(hapi.ComponentTypeConsole)
	}

	flags := internal.FlagSet{}
	_ = flags.Parse(container.Args)
	args := flags.Flags()

	if port, ok := args["-Dserver.port"]; ok {
		defPort, err := a.generateContainerPort(port)
		if err != nil {
			return nil, err
		}
		// since startup params have higher priority than env, so we don't no need to add port to env here.
		container.Ports = extendPorts(container.Ports, defPort)
	} else {
		port, yes := a.hasDefinedPortInEnv(&container)
		if yes {
			defPort, err := a.generateContainerPort(port)
			if err != nil {
				return nil, err
			}
			container.Ports = extendPorts(container.Ports, defPort)
		} else {
			container.Ports = extendPorts(container.Ports, consolePort)
			// add default server port to env if user doesn't define -Dserver.port in args or SERVER_PORT in env
			container.Env = extendEnvs(container.Env, consoleEnvVars...)
		}
	}

	if _, ok := args["-Dplain.hstream.privateAddress"]; !ok {
		hServerContainer := &hdb.Spec.HServer.Container
		ports := coverPortsFromArgs(hServerContainer.Args, extendPorts(hServerContainer.Ports, hServerPort))
		port := int32(0)
		for i := range ports {
			if ports[i].Name == "port" {
				port = ports[i].ContainerPort
			}
		}
		hServerSvc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer)
		address := fmt.Sprintf("%s:%d", hServerSvc.Name, port)

		container.Env = extendEnvs(container.Env, corev1.EnvVar{
			Name:  consoleEnvHServerAddr,
			Value: address,
		})
	}

	return append([]corev1.Container{container}, console.SidecarContainers...), nil
}

func (a addConsole) generateContainerPort(p string) (containerPort corev1.ContainerPort, err error) {
	port, err := strconv.Atoi(p)
	if err != nil {
		err = errors.New("parse console server port failed")
		return
	}

	containerPort = consolePort
	containerPort.ContainerPort = int32(port)
	return
}

func (a addConsole) hasDefinedPortInEnv(container *corev1.Container) (port string, yes bool) {
	for i := range container.Env {
		if container.Env[i].Name == consoleEnvPortName {
			return container.Env[i].Value, true
		}
	}
	return "", false
}
