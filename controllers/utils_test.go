package controllers

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
				ContainerPort: 1,
			},
			{
				Name:          "admin-port",
				ContainerPort: 1,
			},
			{
				Name:          "unknown-port",
				ContainerPort: 2,
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
				ContainerPort: 1,
			},
			{
				Name:          "unset-2",
				ContainerPort: 2,
			},
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

	It("test getHMetaAddr by external HMeta cluster", func() {
		hdb := &hapi.HStreamDB{
			Spec: hapi.HStreamDBSpec{
				ExternalHMeta: &hapi.ExternalHMeta{
					Host:      "rqlite-svc",
					Port:      4001,
					Namespace: "default",
				}},
		}
		addr, err := getHMetaAddr(hdb)
		Expect(err).To(Succeed())
		Expect(addr).To(Equal("rqlite-svc.default:4001"))
	})

	It("test getHMetaAddr by internal HMeta", func() {
		hdb := &hapi.HStreamDB{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hstreamdb-sample",
				Namespace: "default",
			},
			Spec: hapi.HStreamDBSpec{
				HMeta: hapi.Component{
					Replicas: 1,
				},
			},
		}
		addr, err := getHMetaAddr(hdb)
		Expect(err).To(Succeed())
		Expect(addr).To(Equal("hstreamdb-sample-internal-hmeta.default:4001"))
	})

	It("test getHMetaAddr with invalid arg", func() {
		hdb := &hapi.HStreamDB{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hstreamdb-sample",
				Namespace: "default",
			},
			Spec: hapi.HStreamDBSpec{
				HMeta: hapi.Component{
					Replicas: 1,
					Container: hapi.Container{
						Args: []string{"invalid arg"},
					},
				},
			},
		}
		_, err := getHMetaAddr(hdb)
		Expect(err).To(HaveOccurred())
	})
})
