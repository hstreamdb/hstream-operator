package controllers

import (
	"context"
	"strconv"
	"strings"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AddHserver", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	hServer := addHServer{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var sts *appsv1.StatefulSet
		It("should successfully get statefulset", func() {
			var err error
			sts, err = getHServerStatefulSet(hdb)
			Expect(err).To(BeNil())
		})

		When("hserver has been deployed", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newSts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.UID).To(Equal(newSts.UID))
				})
			})
			Context("scale up HServer replicas", func() {
				replicas := int32(3)
				BeforeEach(func() {
					hdb.Spec.HServer.Replicas = replicas
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get the correct number of replicas", func() {
					sts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(*sts.Spec.Replicas).Should(Equal(replicas))
				})

				It("should generate the correct default container command", func() {
					sts, err := getHServerStatefulSet(hdb)

					Expect(err).To(BeNil())

					defaultContainerCommand := []string{"bash", "-c"}
					defaultContainerArgs := []string{
						strings.Join([]string{"/usr/local/bin/hstream-server",
							"--config-path", "/etc/hstream/config.yaml",
							"--bind-address", "0.0.0.0",
							"--advertised-address $(POD_IP)",
							"--store-config", "/etc/logdevice/config.json",
							"--store-admin-host", "hstreamdb-sample-admin-server.default",
							"--metastore-uri", "rq://hstreamdb-sample-internal-hmeta.default:4001",
							"--server-id", "$(hostname | grep -o '[0-9]*$')",
							"--port", "6570",
							"--internal-port", "6571",
							"--seed-nodes", "hstreamdb-sample-hserver-0.hstreamdb-sample-internal-hserver.default:6571,hstreamdb-sample-hserver-1.hstreamdb-sample-internal-hserver.default:6571,hstreamdb-sample-hserver-2.hstreamdb-sample-internal-hserver.default:6571",
						}, " "),
					}

					Expect(sts.Spec.Template.Spec.Containers[0].Command).Should(Equal(defaultContainerCommand))
					Expect(sts.Spec.Template.Spec.Containers[0].Args).Should(Equal(defaultContainerArgs))
				})
			})
			Context("update container name", func() {
				name := "hdb-hserver"
				BeforeEach(func() {
					hdb.Spec.HServer.Container.Name = name
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container name", func() {
					sts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(sts.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
				})
			})
			Context("update container command", func() {
				command := []string{"bash", "-c", "|", "echo 'hello world'"}
				BeforeEach(func() {
					hdb.Spec.HServer.Container.Command = command
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container command", func() {
					sts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(sts.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
				})
			})
			Context("if port is defined in args", func() {
				port := "6571"

				BeforeEach(func() {
					hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
						"--port", port)
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should use the specified port", func() {
					sts, err := getHServerStatefulSet(hdb)

					Expect(err).To(BeNil())

					Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--port %s", port))
					Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ContainElements(
						WithTransform(func(p corev1.ContainerPort) string { return strconv.Itoa(int(p.ContainerPort)) }, Equal(port)),
					))
				})
			})
			Context("define internal-port in args", func() {
				internalPort := "6572"
				BeforeEach(func() {
					hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
						"--internal-port", internalPort)
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
					Expect(requeue).To(BeNil())
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get internal-port", func() {
					sts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--internal-port %s", internalPort))
					Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ContainElements(
						WithTransform(func(p corev1.ContainerPort) string { return strconv.Itoa(int(p.ContainerPort)) }, Equal(internalPort)),
					))
				})
			})
			Context("define log level in args", func() {
				BeforeEach(func() {
					hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
						"--log-level", "debug")
					requeue = hServer.reconcile(ctx, clusterReconciler, hdb)
					Expect(requeue).To(BeNil())
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get internal-port and seeds-node from args", func() {
					sts, err := getHServerStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--log-level debug"))
				})
			})
		})
	})
})

func getHServerStatefulSet(hdb *hapi.HStreamDB) (sts *appsv1.StatefulSet, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHServer.GetResName(hdb.Name),
	}
	sts = &appsv1.StatefulSet{}
	err = k8sClient.Get(context.TODO(), keyObj, sts)
	return
}
