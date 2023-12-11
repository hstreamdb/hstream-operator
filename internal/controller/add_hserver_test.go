package controller

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

var (
	hdb     *hapi.HStreamDB
	hserver addHServer
)

func reconcile() *requeue {
	return hserver.reconcile(ctx, clusterReconciler, hdb)
}

var _ = Describe("controller/add_server", func() {
	hserver = addHServer{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		Expect(reconcile()).To(BeNil())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("after reconcile", func() {
		var sts *appsv1.StatefulSet

		It("should get hserver statefulset successfully", func() {
			var err error
			sts, err = getHServerStatefulSet(hdb)

			Expect(err).To(BeNil())
		})

		When("hserver nodes have been deployed", func() {
			It("reconcile with nothing changed", func() {
				Expect(reconcile()).To(BeNil())

				newSts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.UID).To(Equal(newSts.UID))
			})

			Context("should scale up HServer replicas successfully", Ordered, func() {
				replicas := int32(3)
				var sts *appsv1.StatefulSet

				BeforeAll(func() {
					hdb.Spec.HServer.Replicas = replicas
					reconcile()

					sts, _ = getHServerStatefulSet(hdb)
				})

				It("should get the correct number of replicas", func() {
					Expect(*sts.Spec.Replicas).To(Equal(replicas))
				})

				It("should generate the correct default container command", func() {
					defaultContainerCommand := []string{"bash", "-c"}
					defaultContainerArgs := []string{
						strings.Join([]string{"/usr/local/bin/hstream-server",
							"--config-path", "/etc/hstream/config.yaml",
							"--bind-address", "0.0.0.0",
							"--advertised-address $(POD_NAME).hstreamdb-sample-internal-hserver.default",
							"--store-config", "/etc/logdevice/config.json",
							"--store-admin-host", "hstreamdb-sample-admin-server.default",
							"--metastore-uri", "rq://hstreamdb-sample-internal-hmeta.default:4001",
							"--server-id", "$(hostname | grep -o '[0-9]*$')",
							"--port", "6570",
							"--internal-port", "6571",
							"--seed-nodes", "hstreamdb-sample-hserver-0.hstreamdb-sample-internal-hserver.default:6571,hstreamdb-sample-hserver-1.hstreamdb-sample-internal-hserver.default:6571,hstreamdb-sample-hserver-2.hstreamdb-sample-internal-hserver.default:6571",
						}, " "),
					}

					Expect(sts.Spec.Template.Spec.Containers[0].Command).To(Equal(defaultContainerCommand))
					Expect(sts.Spec.Template.Spec.Containers[0].Args).To(Equal(defaultContainerArgs))
				})
			})

			It("should get the updated container name", func() {
				name := "my-hserver"

				hdb.Spec.HServer.Container.Name = name
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
			})

			It("should get the updated container command", func() {
				command := []string{"bash", "-c", "echo 'hello world'"}

				hdb.Spec.HServer.Container.Command = command
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
			})

			It("should use the specified port", func() {
				port := "6571"

				hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
					"--port", port)
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--port %s", port))
				Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ContainElements(
					WithTransform(func(p corev1.ContainerPort) string { return strconv.Itoa(int(p.ContainerPort)) }, Equal(port)),
				))
			})

			It("should use the specified internal-port", func() {
				internalPort := "6572"

				hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
					"--internal-port", internalPort)
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--internal-port %s", internalPort))
				Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ContainElements(
					WithTransform(func(p corev1.ContainerPort) string { return strconv.Itoa(int(p.ContainerPort)) }, Equal(internalPort)),
				))
			})

			It("should use defined log level", func() {
				hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
					"--log-level", "debug")
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--log-level debug"))
			})

			It("should override the default config successfully", func() {
				hdb.Spec.HServer.Container.Args = append(hdb.Spec.HServer.Container.Args,
					"--config-path", "/etc/custom/hstream/config")
				Expect(reconcile()).To(BeNil())

				sts, err := getHServerStatefulSet(hdb)

				Expect(err).To(BeNil())
				Expect(sts.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("--config-path /data/custom/hstream/config-override.yaml"))
				Expect(sts.Spec.Template.Spec.InitContainers).To(HaveLen(2))
			})
		})
	})
})

func getHServerStatefulSet(hdb *hapi.HStreamDB) (sts *appsv1.StatefulSet, err error) {
	sts = &appsv1.StatefulSet{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHServer.GetResName(hdb.Name),
	}, sts)

	return
}
