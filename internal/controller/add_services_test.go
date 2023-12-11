package controller

import (
	"context"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("AddServices", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	addServices := addServices{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = addServices.reconcile(ctx, hstreamdbReconciler, hdb)
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		//var hstore, hserver, adminServer *corev1.Service
		var hstore *corev1.Service
		It("should successfully get services", func() {
			var err error
			hstore, err = getHeadlessService(hdb, hapi.ComponentTypeHStore)
			Expect(err).To(BeNil())
			_, err = getHeadlessService(hdb, hapi.ComponentTypeHServer)
			Expect(err).To(BeNil())
			_, err = getService(hdb, hapi.ComponentTypeAdminServer)
			Expect(err).To(BeNil())
			_, err = getHeadlessService(hdb, hapi.ComponentTypeHMeta)
			Expect(err).To(BeNil())
		})

		When("services has been deployed", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = addServices.reconcile(ctx, hstreamdbReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newHStore, err := getHeadlessService(hdb, hapi.ComponentTypeHStore)
					Expect(err).To(BeNil())
					Expect(hstore.UID).To(Equal(newHStore.UID))
				})
			})

			Context("update service port", func() {
				BeforeEach(func() {
					hdb.Spec.HStore.Container.Ports = []corev1.ContainerPort{
						{
							Name:          "user-defined-port",
							ContainerPort: 5440,
						},
					}
					requeue = addServices.reconcile(ctx, hstreamdbReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new service name", func() {
					svc, err := getHeadlessService(hdb, hapi.ComponentTypeHStore)
					Expect(err).To(BeNil())
					Expect(svc.Spec.Ports).To(ContainElement(corev1.ServicePort{
						Name:     "user-defined-port",
						Protocol: corev1.ProtocolTCP,
						Port:     5440,
						TargetPort: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 5440,
						},
					}))
				})
			})

			Context("check hmete headless service", func() {
				It("PublishNotReadyAddresses should be true", func() {
					svc, err := getHeadlessService(hdb, hapi.ComponentTypeHMeta)
					Expect(err).To(BeNil())
					Expect(svc.Spec.PublishNotReadyAddresses).To(BeTrue())
				})
			})
		})
	})
})

func getService(hdb *hapi.HStreamDB, compType hapi.ComponentType) (svc *corev1.Service, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      compType.GetResName(hdb.Name),
	}
	svc = &corev1.Service{}
	err = k8sClient.Get(context.TODO(), keyObj, svc)
	return
}

func getHeadlessService(hdb *hapi.HStreamDB, compType hapi.ComponentType) (svc *corev1.Service, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      internal.GetResNameOnPanic(hdb, "internal-"+string(compType)),
	}
	svc = &corev1.Service{}
	err = k8sClient.Get(context.TODO(), keyObj, svc)
	return
}
