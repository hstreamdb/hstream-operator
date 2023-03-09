package controllers

import (
	"context"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AddAdminServer", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	addAdminServer := addAdminServer{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())

		requeue = addAdminServer.reconcile(ctx, clusterReconciler, hdb)
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
			deploy, err = getAdminServerDeployment(hdb)
			Expect(err).To(BeNil())
		})

		When("admin server has been deploy", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = addAdminServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newDeploy, err := getAdminServerDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.UID).To(Equal(newDeploy.UID))
				})
			})

			Context("update container name", func() {
				name := "hdb-admin-server"
				BeforeEach(func() {
					hdb.Spec.AdminServer.Container.Name = name
					requeue = addAdminServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container name", func() {
					deploy, err := getAdminServerDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
				})
			})
			Context("update container command", func() {
				command := []string{"bash", "-c", "|", "echo 'hello world'"}
				BeforeEach(func() {
					hdb.Spec.AdminServer.Container.Command = command
					requeue = addAdminServer.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container command", func() {
					deploy, err := getAdminServerDeployment(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
				})
			})
		})
	})
})

func getAdminServerDeployment(hdb *hapi.HStreamDB) (deploy *appsv1.Deployment, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeAdminServer.GetResName(hdb.Name),
	}
	deploy = &appsv1.Deployment{}
	err = k8sClient.Get(context.TODO(), keyObj, deploy)
	return
}
