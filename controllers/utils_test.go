package controllers

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Utils", func() {
	It("test structAssign", func() {
		type User struct {
			Name string
			Age  int
		}
		var u1 = User{
			Name: "name",
			Age:  1,
		}
		var u2 User

		structAssign(&u2, &u1)
		Expect(u2).To(Equal(u1))
	})

	It("test mergeMap", func() {
		m1 := map[string]string{
			"a": "1",
		}
		m2 := map[string]string{
			"a": "2",
			"b": "2",
		}
		mergeMap(m1, m2)
		Expect(m1).To(BeComparableTo(map[string]string{
			"a": "2",
			"b": "2",
		}))
	})

	It("test extendEnv", func() {
		container := &corev1.Container{
			Env: []corev1.EnvVar{
				{
					Name:  "a",
					Value: "1",
				},
				{
					Name:  "c",
					Value: "1",
				},
			},
		}
		newEnv := []corev1.EnvVar{
			{
				Name:  "a",
				Value: "2",
			},
			{
				Name:  "b",
				Value: "2",
			},
		}
		extendEnv(container, newEnv)
		Expect(container.Env).To(ContainElements([]corev1.EnvVar{
			{
				Name:  "a",
				Value: "1",
			},
			{
				Name:  "b",
				Value: "2",
			},
			{
				Name:  "c",
				Value: "1",
			},
		}))
	})

	It("test extendArg", func() {
		container := &corev1.Container{
			Args: []string{
				"--a", "1",
				"--b", "1",
			},
		}
		newArgs := map[string]string{
			"--a": "2",
			"--c": "2",
		}
		err := extendArg(container, newArgs)
		Expect(err).To(BeNil())
		Expect(container.Args).To(ContainElements("--a", "1", "--b", "1", "--c", "2"))
	})

	It("test mergePorts", func() {
		required := []corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}
		userDefined := []corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 2,
			},
			{
				Name:          "unknown-port",
				ContainerPort: 2,
			},
		}
		newPorts := mergePorts(required, userDefined)
		Expect(newPorts).To(Equal([]corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 2,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}))
	})

	It("test mergePorts2", func() {
		required := []corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}
		userDefined := []corev1.ContainerPort{
			{
				Name:          "admin-port",
				ContainerPort: 2,
			},
		}
		newPorts := mergePorts(required, userDefined)
		Expect(newPorts).To(Equal([]corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 2,
			},
		}))
	})

	It("test mergePorts3", func() {
		required := []corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 1,
			},
			{
				Name:          "gossip-port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}
		userDefined := []corev1.ContainerPort{
			{
				Name:          "gossip-port",
				ContainerPort: 2,
			},
		}
		newPorts := mergePorts(required, userDefined)
		Expect(newPorts).To(Equal([]corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 1,
			},
			{
				Name:          "gossip-port",
				ContainerPort: 2,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}))
	})

	Context("test use pvc", func() {
		var hdb *appsv1alpha1.HStreamDB
		BeforeEach(func() {
			hdb = &appsv1alpha1.HStreamDB{
				Spec: appsv1alpha1.HStreamDBSpec{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			}
		})

		It("should use pvc", func() {
			Expect(usePvc(hdb)).To(BeTrue())
		})

		It("should not user pvc", func() {
			hdb.Spec.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("0Gi")
			Expect(usePvc(hdb)).To(BeFalse())
		})

		It("should not user pvc", func() {
			delete(hdb.Spec.VolumeClaimTemplate.Spec.Resources.Requests, corev1.ResourceStorage)
			Expect(usePvc(hdb)).To(BeFalse())
		})
	})

	Context("test obj hash", func() {
		var obj1, obj2 *metav1.ObjectMeta
		BeforeEach(func() {
			obj1 = &metav1.ObjectMeta{
				Annotations: map[string]string{
					appsv1alpha1.LastSpecKey: "hash1",
				},
			}
			obj2 = &metav1.ObjectMeta{
				Annotations: map[string]string{
					appsv1alpha1.LastSpecKey: "hash1",
				},
			}
		})

		It("hash don't changed", func() {
			Expect(isHashChanged(obj1, obj2)).To(BeFalse())
		})

		It("hash changed", func() {
			obj1.Annotations[appsv1alpha1.LastSpecKey] = "hash3"
			Expect(isHashChanged(obj1, obj2)).To(BeTrue())
		})
	})
})
