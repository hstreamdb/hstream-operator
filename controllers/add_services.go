package controllers

import (
	"context"
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addServices struct{}

func (a addServices) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	var err error
	if err = a.addHMetaService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	if err = a.addAdminServerService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	if err = a.addHStoreService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	if err = a.addHServerService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addServices) addHServerService(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) (err error) {
	hserver := hdb.Spec.HServer
	ports, err := getPorts(&hserver.Container, []corev1.ContainerPort{
		hServerPort, hServerInternalPort,
	})
	if err != nil {
		return fmt.Errorf("parse hServer args failed. %w", err)
	}

	service := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer, ports...)
	service.Spec.PublishNotReadyAddresses = true
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addHStoreService(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) (err error) {
	ports, err := getPorts(&hdb.Spec.HStore.Container, hStorePorts)
	if err != nil {
		return fmt.Errorf("parse hStore args failed. %w", err)
	}
	service := internal.GetHeadlessService(hdb, hapi.ComponentTypeHStore, ports...)
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addAdminServerService(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) (err error) {
	ports, err := getPorts(&hdb.Spec.AdminServer.Container, adminServerPorts)
	if err != nil {
		return fmt.Errorf("parse adminServer args failed. %w", err)
	}
	service := internal.GetService(hdb, hapi.ComponentTypeAdminServer, ports...)
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addHMetaService(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB) (err error) {
	if hdb.Spec.ExternalHMeta != nil {
		return nil
	}

	hmeta := hdb.Spec.HMeta

	flags := internal.FlagSet{}
	if err = flags.Parse(hmeta.Container.Args); err != nil {
		return fmt.Errorf("parse hmeta args failed. %w", err)
	}

	parsedArgs := flags.Flags()
	ports := getHMetaContainerPorts(&hmeta.Container, parsedArgs)
	servicePorts := convertToServicePort(ports)

	service := internal.GetHeadlessService(hdb, hapi.ComponentTypeHMeta, servicePorts...)
	service.Spec.PublishNotReadyAddresses = true
	return a.createOrUpdate(ctx, r, hdb, &service)
}

// createOrUpdate create or updates selected safe fields on a service based on a new
// service definition.
func (a addServices) createOrUpdate(ctx context.Context, r *HStreamDBReconciler, hdb *hapi.HStreamDB,
	newService *corev1.Service) (err error) {

	newService.Annotations[hapi.LastSpecKey] = internal.GetObjectHash(newService)

	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "service", newService.Name)

	existingService := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKeyFromObject(newService), existingService)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return
		}

		logger.Info("Creating service")
		if err = ctrl.SetControllerReference(hdb, newService, r.Scheme); err != nil {
			return err
		}
		return r.Create(ctx, newService)
	}
	if !isHashChanged(&existingService.ObjectMeta, &newService.ObjectMeta) {
		return nil
	}

	metadata := existingService.ObjectMeta
	_ = mergeLabelsInMetadata(&metadata, newService.ObjectMeta)
	_ = mergeAnnotations(&metadata, newService.ObjectMeta)
	existingService.ObjectMeta = metadata
	existingService.Spec = newService.Spec

	logger.Info("Updating service")
	return r.Update(ctx, existingService)
}

func getPorts(container *hapi.Container, defaultPorts []corev1.ContainerPort) (
	[]corev1.ServicePort, error) {

	flags := internal.FlagSet{}
	if err := flags.Parse(container.Args); err != nil {
		return nil, err
	}

	args := flags.Flags()
	ports := extendPorts(args, container.Ports, defaultPorts)
	return convertToServicePort(ports), nil
}

func convertToServicePort(ports []corev1.ContainerPort) []corev1.ServicePort {
	var servicePorts []corev1.ServicePort
	for _, port := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
	}
	return servicePorts
}
