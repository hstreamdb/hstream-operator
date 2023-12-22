package controller

import (
	"context"
	"fmt"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// this test case requires to connect to the running k8s cluster in local or anywhere
// set env: export USE_EXISTING_CLUSTER=true
// run the test: ginkgo run --label-filter 'k8s' controllers/
var _ = Describe("AddConsole", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	addConsole := addConsole{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		hdb.Spec.Console = &hapi.Component{
			Image:           "hstreamdb/hstream-console",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Replicas:        1,
		}
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var deploy *appsv1.Deployment
		It("should successfully get deployment", func() {
			var err error
			deploy, err = getConsoleDeployment(hdb)
			Expect(err).To(BeNil())
		})

		When("console has been deployed", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newDeploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.UID).To(Equal(newDeploy.UID))
				})
			})

			Context("should contain default config", func() {
				It("should get default env", func() {
					hServerContainer := &hdb.Spec.HServer.Container
					ports := coverPortsFromArgs(hServerContainer.Args, extendPorts(hServerContainer.Ports, constants.DefaultHServerPort))
					port := int32(0)
					for i := range ports {
						if ports[i].Name == "port" {
							port = ports[i].ContainerPort
						}
					}
					hServerSvc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHServer)
					address := fmt.Sprintf("%s:%d", hServerSvc.Name, port)

					defEnvVars := make([]corev1.EnvVar, len(consoleEnvVars))
					copy(defEnvVars, consoleEnvVars)
					defEnvVars = append(defEnvVars, corev1.EnvVar{
						Name:  consoleEnvHServerAddr,
						Value: address,
					})

					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Env).Should(ContainElements(defEnvVars))
				})
			})

			Context("update container name", func() {
				name := "hdb-console"
				BeforeEach(func() {
					hdb.Spec.Console.Container.Name = name
					requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container name", func() {
					deploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
				})
			})
			Context("update container command", func() {
				command := []string{"bash", "-c", "|", "echo 'hello world'"}
				BeforeEach(func() {
					hdb.Spec.Console.Container.Command = command
					requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container command", func() {
					deploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
				})
			})

			Context("update container env", func() {
				BeforeEach(func() {
					hdb.Spec.Console.Container.Env = []corev1.EnvVar{
						{
							Name:  "SERVER_PORT",
							Value: "5178",
						},
					}
					requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container port", func() {
					deploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Ports).Should(ContainElement(corev1.ContainerPort{
						Name:          "server-port",
						ContainerPort: 5178,
						Protocol:      corev1.ProtocolTCP,
					}))
				})
			})

			Context("update container args", func() {
				BeforeEach(func() {
					hdb.Spec.Console.Container.Args = []string{
						"-Dserver.port=5179",
						"-Dplain.hstream.privateAddress=hstreamdb-sample-internal-hserver:6570",
					}
					requeue = addConsole.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container port", func() {
					deploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Ports).Should(ContainElement(corev1.ContainerPort{
						Name:          "server-port",
						ContainerPort: 5179,
						Protocol:      corev1.ProtocolTCP,
					}))
				})
				It("should not contain port and privateAddress in the env", func() {
					deploy, err := getConsoleDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					for _, env := range deploy.Spec.Template.Spec.Containers[0].Env {
						Expect(env.Name).ShouldNot(Equal(consoleEnvPortName))
						Expect(env.Name).ShouldNot(Equal(consoleEnvHServerAddr))
					}
				})
			})
		})
	})
})

func getConsoleDeployment(hdb *hapi.HStreamDB) (deploy *appsv1.Deployment, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeConsole.GetResName(hdb),
	}
	deploy = &appsv1.Deployment{}
	err = k8sClient.Get(context.TODO(), keyObj, deploy)
	return
}
