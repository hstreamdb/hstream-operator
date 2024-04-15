package controller

import (
	"context"
	"fmt"

	"github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addServices struct{}

func (a addServices) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB) *requeue {
	var err error
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

func (a addServices) addHServerService(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB) (err error) {
	hserver := &hdb.Spec.HServer
	var defaultPorts []corev1.ContainerPort

	if hdb.Spec.Config.KafkaMode {
		defaultPorts = constants.DefaultKafkaHServerPorts
	} else {
		defaultPorts = constants.DefaultHServerPorts
	}

	ports, err := getPorts(&hserver.Container, defaultPorts...)
	if err != nil {
		return fmt.Errorf("parse HServer args failed. %w", err)
	}

	service := v1alpha2.ComponentTypeHServer.GetHeadlessService(hdb, ports)
	if err = a.createOrUpdate(ctx, r, hdb, &service); err != nil {
		return
	}

	if hdb.Spec.Config.ExternalAccess != nil && hdb.Spec.Config.ExternalAccess.AccessType == v1alpha2.AccessTypeNodePort {
		nodeportServices := v1alpha2.ComponentTypeHServer.GetNodePortServices(hdb, ports)

		for _, s := range nodeportServices {
			err = a.createOrUpdate(ctx, r, hdb, &s)
			if err != nil {
				return
			}
		}
	}

	return
}

func (a addServices) addHStoreService(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB) (err error) {
	ports, err := getPorts(&hdb.Spec.HStore.Container, constants.DefaultHStorePorts...)
	if err != nil {
		return fmt.Errorf("parse HStore args failed. %w", err)
	}
	service := v1alpha2.ComponentTypeHStore.GetHeadlessService(hdb, ports)
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addAdminServerService(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB) (err error) {
	ports, err := getPorts(&hdb.Spec.AdminServer.Container, constants.DefaultAdminServerPort)
	if err != nil {
		return fmt.Errorf("parse AdminServer args failed. %w", err)
	}
	service := v1alpha2.ComponentTypeAdminServer.GetService(hdb, ports)
	return a.createOrUpdate(ctx, r, hdb, &service)
}

// createOrUpdate create or updates selected safe fields on a service based on a new
// service definition.
func (a addServices) createOrUpdate(ctx context.Context, r *HStreamDBReconciler, hdb *v1alpha2.HStreamDB,
	newService *corev1.Service) (err error) {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "service", newService.Name)

	newService.Annotations[v1alpha2.LastSpecKey] = internal.GetObjectHash(newService)

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

func getPorts(container *corev1.Container, defaultPorts ...corev1.ContainerPort) ([]corev1.ServicePort, error) {
	ports := coverPortsFromArgs(container.Args, extendPorts(container.Ports, defaultPorts...))
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
