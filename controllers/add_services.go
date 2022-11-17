package controllers

import (
	"context"
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
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
	ports := mergePorts(hServerPorts, hServer.Container.Ports)

	var sPort string
	for _, port := range ports {
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
		if port.Name == "port" {
			sPort = strconv.Itoa(int(port.ContainerPort))
		}
	}

	if sPort == "" {
		return errors.New("invalid port")
	}

	/*hdb.Status.HServersAddr = fmt.Sprintf("%s-0.%s.%s:%s",
	appsv1alpha1.ComponentTypeHServer.GetResName(hdb.Name),
	service.Name,
	service.Namespace,
	sPort)*/
	return a.createOrUpdate(ctx, r, hdb, &service)
}

func (a addServices) addHStoreService(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB) (err error) {
	service := internal.GetHeadlessService(hdb, appsv1alpha1.ComponentTypeHStore)

	hStore := &hdb.Spec.HStore
	ports := mergePorts(hStorePorts, hStore.Container.Ports)

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
	ports := mergePorts(adminServerPorts, adminServer.Container.Ports)

	var sPort string
	var servicePorts []corev1.ServicePort
	for _, port := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     port.Name,
			Protocol: port.Protocol,
			Port:     port.ContainerPort,
		})
		if port.Name == "port" {
			sPort = strconv.Itoa(int(port.ContainerPort))
		}
	}

	if sPort == "" {
		return errors.New("invalid port")
	}

	service := internal.GetService(hdb, servicePorts, appsv1alpha1.ComponentTypeAdminServer)

	//hdb.Status.AdminServerAddr = fmt.Sprintf("%s.%s:%s", service.Name, service.Namespace, sPort)
	return a.createOrUpdate(ctx, r, hdb, &service)
}

// createOrUpdate create or updates selected safe fields on a service based on a new
// service definition.
func (a addServices) createOrUpdate(ctx context.Context, r *HStreamDBReconciler, hdb *appsv1alpha1.HStreamDB,
	newService *corev1.Service) (err error) {

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

	originalSpec := existingService.Spec.DeepCopy()
	existingService.Spec.Selector = originalSpec.Selector

	needUpdate := !equality.Semantic.DeepEqual(existingService.Spec, *originalSpec)
	metadata := existingService.ObjectMeta
	if mergeLabelsInMetadata(&metadata, newService.ObjectMeta) {
		needUpdate = true
	}
	if mergeAnnotations(&metadata, newService.ObjectMeta) {
		needUpdate = true
	}
	if needUpdate {
		existingService.ObjectMeta = metadata
		logger.Info("Updating service")
		return r.Update(ctx, existingService)
	}
	return
}
