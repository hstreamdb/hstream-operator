package controller

import (
	"context"
	"fmt"
	"strconv"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AddHMeta", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	hmeta := addHMeta{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		BeforeEach(func() {
			requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
		})
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var sts *appsv1.StatefulSet
		It("should successfully get sts", func() {
			var err error
			sts, err = getHMetaStatefulSet(hdb)
			Expect(err).To(BeNil())
		})

		Context("check container arg", func() {
			flag := internal.FlagSet{}
			var err error
			BeforeEach(func() {
				err = flag.Parse(sts.Spec.Template.Spec.Containers[0].Args)
				Expect(err).To(BeNil())
			})

			It("bootstrap-expect should be equal to replica", func() {
				Expect(flag.Flags()).Should(HaveKeyWithValue("--bootstrap-expect", strconv.Itoa(int(hdb.Spec.HMeta.Replicas))))
			})

			It("disco-mode should be dns", func() {
				Expect(flag.Flags()).Should(HaveKeyWithValue("--bootstrap-expect", strconv.Itoa(int(hdb.Spec.HMeta.Replicas))))
			})

			It("disco-config should be internal-svc name", func() {
				svc := internal.GetHeadlessService(hdb, hapi.ComponentTypeHMeta)
				Expect(flag.Flags()).Should(HaveKeyWithValue("--disco-config",
					fmt.Sprintf(`{"name":"%s"}`, svc.Name)))
			})
		})

		When("hmeta has been deployed", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newSts, err := getHMetaStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.UID).To(Equal(newSts.UID))
				})
			})

			Context("update container name", func() {
				name := "hdb-hmeta"
				BeforeEach(func() {
					hdb.Spec.HMeta.Container.Name = name
					requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container name", func() {
					deploy, err := getHMetaStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(deploy.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(deploy.Spec.Template.Spec.Containers[0].Name).To(Equal(name))
				})
			})
			Context("update container command", func() {
				command := []string{"bash", "-c", "|", "echo 'hello world'"}
				BeforeEach(func() {
					hdb.Spec.HMeta.Container.Command = command
					requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get new container command", func() {
					sts, err := getHMetaStatefulSet(hdb)
					Expect(err).To(BeNil())
					Expect(sts.Spec.Template.Spec.Containers).NotTo(BeEmpty())
					Expect(sts.Spec.Template.Spec.Containers[0].Command).To(Equal(command))
				})
			})
		})
	})

	Context("with a reconciled cluster that specify http-addr", func() {
		httpAddr := "localhost:4002"
		var sts *appsv1.StatefulSet
		var err error
		BeforeEach(func() {
			hdb.Spec.HMeta.Container.Args = []string{
				"--http-addr",
				httpAddr,
			}
			requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
			Expect(requeue).To(BeNil())

			sts, err = getHMetaStatefulSet(hdb)
			Expect(err).To(BeNil())
		})

		Context("check container arg", func() {
			flag := internal.FlagSet{}
			BeforeEach(func() {
				err = flag.Parse(sts.Spec.Template.Spec.Containers[0].Args)
				Expect(err).To(BeNil())
			})

			It("http-addr should be equal to \"localhost:4002\"", func() {
				Expect(flag.Flags()).Should(HaveKeyWithValue("--http-addr", httpAddr))
			})
		})

		Context("check container port", func() {
			It("port name should be \"port\"", func() {
				Expect(sts.Spec.Template.Spec.Containers[0].Ports[0].Name).Should(Equal("port"))
			})
			It("port should be 4002", func() {
				Expect(sts.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).Should(Equal(int32(4002)))
			})
		})
	})

	Context("with a reconciled cluster that used pvc", func() {
		storageClassName := "standard"
		BeforeEach(func() {
			sts, err := getHMetaStatefulSet(hdb)
			if err != nil {
				Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
			} else {
				_ = k8sClient.Delete(ctx, sts)
			}

			hdb.Spec.HMeta.VolumeClaimTemplate = &corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		It("should get new sts with pvc", func() {
			sts, err := getHMetaStatefulSet(hdb)
			Expect(err).To(BeNil())
			Expect(sts.Spec.VolumeClaimTemplates).To(HaveLen(1))
			Expect(*sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(Equal(storageClassName))
			Expect(sts.Spec.VolumeClaimTemplates[0].Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))

		})
	})

	Context("use external HMeta cluster", func() {
		BeforeEach(func() {
			hdb.Spec.ExternalHMeta = &hapi.ExternalHMeta{
				Host:      "rqlite-svc",
				Port:      4001,
				Namespace: "default",
			}
			sts, err := getHMetaStatefulSet(hdb)
			if err != nil {
				Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
			} else {
				_ = k8sClient.Delete(ctx, sts)
			}
			requeue = hmeta.reconcile(ctx, hstreamdbReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		It("should not get sts", func() {
			_, err := getHMetaStatefulSet(hdb)
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

func getHMetaStatefulSet(hdb *hapi.HStreamDB) (sts *appsv1.StatefulSet, err error) {
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHMeta.GetResName(hdb.Name),
	}
	sts = &appsv1.StatefulSet{}
	err = k8sClient.Get(context.TODO(), keyObj, sts)
	return
}
