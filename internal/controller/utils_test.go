package controller

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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

	It("test extendEnvs", func() {
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

		Eventually(extendEnvs(container.Env, newEnv...)).Should(ContainElements(
			[]corev1.EnvVar{
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
			},
		))
	})

	It("test extendArg", func() {
		container := &corev1.Container{
			Args: []string{
				"--a", "1",
				"--b", "1",
			},
		}
		defaultArgs := []string{
			"--a", "2",
			"--c", "2",
		}
		args, _ := extendArgs(container.Args, defaultArgs...)
		Expect(args).To(Equal([]string{
			"--a", "1",
			"--b", "1",
			"--c", "2",
		}))
	})

	It("test coverPorts", func() {
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
		userDefinedArgs := []string{
			"--unknown-port", "1",
			"--port", "2",
		}
		newPorts := coverPortsFromArgs(userDefinedArgs, required)
		Expect(newPorts).To(Equal([]corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 2,
			},
			{
				Name:          "gossip-port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
		}))
	})

	It("test extendPorts", func() {
		container := &corev1.Container{
			Ports: []corev1.ContainerPort{
				{
					Name:          "unknown-port",
					ContainerPort: 1,
				},
				{
					Name:          "port",
					ContainerPort: 1,
				},
			},
		}
		args := []string{
			"--port", "2",
		}
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
		container.Ports = coverPortsFromArgs(args, extendPorts(container.Ports, required...))
		Expect(container.Ports).To(ConsistOf([]corev1.ContainerPort{
			{
				Name:          "port",
				ContainerPort: 2,
			},
			{
				Name:          "gossip-port",
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
			{
				Name:          "unknown-port",
				ContainerPort: 1,
			},
		}))
	})

	Context("test obj hash", func() {
		var obj1, obj2 *metav1.ObjectMeta
		BeforeEach(func() {
			obj1 = &metav1.ObjectMeta{
				Annotations: map[string]string{
					hapi.LastSpecKey: "hash1",
				},
			}
			obj2 = &metav1.ObjectMeta{
				Annotations: map[string]string{
					hapi.LastSpecKey: "hash1",
				},
			}
		})

		It("hash don't changed", func() {
			Expect(isHashChanged(obj1, obj2)).To(BeFalse())
		})

		It("hash changed", func() {
			obj1.Annotations[hapi.LastSpecKey] = "hash3"
			Expect(isHashChanged(obj1, obj2)).To(BeTrue())
		})
	})
})
