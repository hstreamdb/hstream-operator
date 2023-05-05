package controllers

import (
	"context"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// this test case requires to connect to the running k8s cluster in local or anywhere
// set env: export USE_EXISTING_CLUSTER=true
// run the test: ginkgo run --label-filter 'k8s' controllers/
var _ = Describe("AddGateway", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	addGateway := addGateway{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	It("should not create gateway if no gateway pointer", func() {
		By("reconcile")
		requeue = addGateway.reconcile(ctx, clusterReconciler, hdb)
		Expect(requeue).To(BeNil())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      hdb.Name + "-gateway",
				Namespace: hdb.Namespace,
			}, &appsv1.Deployment{})
			return k8sErrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	When("gateway pointer is set, but hserver not ready", func() {
		JustBeforeEach(func() {
			gateway := &hapi.Gateway{}
			gateway.Endpoint = "localhost"
			gateway.Port = 14789
			gateway.Image = "hstreamdb/hstream-gateway:latest"
			gateway.Replicas = 1

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).Should(Succeed())
			hdb.Spec.Gateway = gateway
			Expect(k8sClient.Update(ctx, hdb.DeepCopy())).Should(Succeed())
		})

		It("should not create gateway if hserver not ready", func() {
			By("reconcile")
			requeue = addGateway.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue.curError).To(BeNil())
			Expect(requeue.message).NotTo(BeEmpty())
			Expect(requeue.delay).To(Equal(10 * time.Second))

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      hdb.Name + "-gateway",
					Namespace: hdb.Namespace,
				}, &appsv1.Deployment{})
				return k8sErrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})

	When("gateway pointer is set, and hserver is ready, not enable mTLS", func() {
		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, getFakePod(hdb))).Should(Succeed())

			gateway := &hapi.Gateway{}
			gateway.Endpoint = "localhost"
			gateway.Port = 14789
			gateway.Image = "hstreamdb/hstream-gateway:latest"
			gateway.Replicas = 1

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).Should(Succeed())
			hdb.Spec.Gateway = gateway
			Expect(k8sClient.Update(ctx, hdb.DeepCopy())).Should(Succeed())

			hdb.Status.HServer.Bootstrapped = true
			Expect(k8sClient.Status().Patch(ctx, hdb.DeepCopy(), client.MergeFrom(hdb))).Should(Succeed())
		})

		JustAfterEach(func() {
			Expect(k8sClient.Delete(ctx, getFakePod(hdb))).Should(Succeed())
		})

		It("should create gateway, but secret is not mount", func() {
			By("reconcile")
			requeue = addGateway.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue).To(BeNil())

			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      hdb.Name + "-gateway",
					Namespace: hdb.Namespace,
				}, deployment)
			}).Should(Succeed())

			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Name).To(Equal(string(hapi.ComponentTypeGateway)))
			Expect(container.Image).To(Equal("hstreamdb/hstream-gateway:latest"))
			Expect(container.Ports).To(ContainElement(corev1.ContainerPort{
				Name:          "port",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 14789,
			}))
			Expect(container.Env).To(ContainElements(
				corev1.EnvVar{Name: "ENDPOINT_HOST", Value: "localhost"},
				corev1.EnvVar{Name: "HSTREAM_SERVICE_URL", Value: "hstream://hstreamdb-sample-internal-hserver.default:6570"},
				corev1.EnvVar{Name: "ENABLE_TLS", Value: "false"},
			))
			Expect(container.VolumeMounts).To(BeEmpty())
			Expect(deployment.Spec.Template.Spec.Volumes).To(BeEmpty())
		})
	})

	When("gateway pointer is set, and hserver is ready, enable mTLS", func() {
		JustBeforeEach(func() {
			Expect(k8sClient.Create(ctx, getFakePod(hdb))).Should(Succeed())

			gateway := &hapi.Gateway{}
			gateway.Endpoint = "localhost"
			gateway.Port = 14789
			gateway.Image = "hstreamdb/hstream-gateway:latest"
			gateway.Replicas = 1
			gateway.SecretRef = &corev1.LocalObjectReference{
				Name: "fake-secret",
			}

			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hdb), hdb)).Should(Succeed())
			hdb.Spec.Gateway = gateway
			Expect(k8sClient.Update(ctx, hdb.DeepCopy())).Should(Succeed())

			hdb.Status.HServer.Bootstrapped = true
			Expect(k8sClient.Status().Patch(ctx, hdb.DeepCopy(), client.MergeFrom(hdb))).Should(Succeed())
		})

		JustAfterEach(func() {
			Expect(k8sClient.Delete(ctx, getFakePod(hdb))).Should(Succeed())
		})

		It("should create gateway, and secret is mount", func() {
			By("reconcile")
			requeue = addGateway.reconcile(ctx, clusterReconciler, hdb)
			Expect(requeue).To(BeNil())

			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      hdb.Name + "-gateway",
					Namespace: hdb.Namespace,
				}, deployment)
			}).Should(Succeed())

			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Name).To(Equal(string(hapi.ComponentTypeGateway)))
			Expect(container.Image).To(Equal("hstreamdb/hstream-gateway:latest"))
			Expect(container.Ports).To(ContainElement(corev1.ContainerPort{
				Name:          "port",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 14789,
			}))
			Expect(container.Env).To(ContainElements(
				corev1.EnvVar{Name: "ENDPOINT_HOST", Value: "localhost"},
				corev1.EnvVar{Name: "HSTREAM_SERVICE_URL", Value: "hstream://hstreamdb-sample-internal-hserver.default:6570"},
				corev1.EnvVar{Name: "ENABLE_TLS", Value: "true"},
				corev1.EnvVar{Name: "TLS_KEY_PATH", Value: "/certs/tls.key"},
				corev1.EnvVar{Name: "TLS_CERT_PATH", Value: "/certs/tls.crt"},
				corev1.EnvVar{Name: "TLS_CA_PATH", Value: "/certs/ca.crt"},
			))
			Expect(container.VolumeMounts).To(ContainElements(
				corev1.VolumeMount{Name: "cert", MountPath: "/certs/tls.key", SubPath: "tls.key"},
				corev1.VolumeMount{Name: "cert", MountPath: "/certs/tls.crt", SubPath: "tls.crt"},
				corev1.VolumeMount{Name: "cert", MountPath: "/certs/ca.crt", SubPath: "ca.crt"},
			))
			Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(
				corev1.Volume{Name: "cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  "fake-secret",
							DefaultMode: pointer.Int32(420),
						},
					},
				},
			))
		})
	})
})

func getFakePod(hdb *hapi.HStreamDB) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hdb.Namespace,
			Name:      hdb.Name + "-hserver",
			Labels: map[string]string{
				hapi.InstanceKey:  hdb.Name,
				hapi.ComponentKey: string(hapi.ComponentTypeHServer),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "hserver",
					Image: "hstreamdb/hstream-hserver:latest",
				},
			},
		},
	}
}
