package controllers

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addServices struct{}

func (a addServices) reconcile(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) *requeue {
	var err error
	if err = a.addHStoreService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	if err = a.addAdminServerService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	if err = a.addHServerService(ctx, r, hdb); err != nil {
		return &requeue{curError: err}
	}
	return nil
}

func (a addServices) addHServerService(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) (err error) {
	service := internal.GetHeadlessService(hdb, appsv1alpha1.ComponentTypeHServer)

	hServer := &hdb.Spec.HServer
	ports, err := a.getPorts(&hServer.Container, hServerPorts)
	if err != nil {
		return fmt.Errorf("parse hServer args failed. %w", err)
	}

	for _, port := range ports {
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
	}
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addHStoreService(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) (err error) {
	service := internal.GetHeadlessService(hdb, appsv1alpha1.ComponentTypeHStore)

	hStore := &hdb.Spec.HStore
	ports, err := a.getPorts(&hStore.Container, hStorePorts)
	if err != nil {
		return fmt.Errorf("parse hStore args failed. %w", err)
	}

	for _, port := range ports {
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
	}
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addAdminServerService(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) (err error) {
	adminServer := &hdb.Spec.AdminServer
	ports, err := a.getPorts(&adminServer.Container, adminServerPorts)
	if err != nil {
		return fmt.Errorf("parse adminServer args failed. %w", err)
	}

	var servicePorts []corev1.ServicePort
	for _, port := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
	}
	service := internal.GetService(hdb, servicePorts, appsv1alpha1.ComponentTypeAdminServer)

	return a.createOrUpdate(ctx, r, hdb, &service)
}

// createOrUpdate create or updates selected safe fields on a service based on a new
// service definition.
func (a addServices) createOrUpdate(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB,
	newService *corev1.Service) (err error) {

	newService.Annotations[appsv1alpha1.LastSpecKey] = internal.GetObjectHash(newService)

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

func (a addServices) getPorts(container *appsv1alpha1.Container, defaultPorts []corev1.ContainerPort) (
	[]corev1.ContainerPort, error) {

	flags := internal.FlagSet{}
	if err := flags.Parse(container.Args); err != nil {
		return nil, err
	}

	args := flags.Flags()
	ports := extendPorts(args, container.Ports, defaultPorts)
	return ports, nil

}
