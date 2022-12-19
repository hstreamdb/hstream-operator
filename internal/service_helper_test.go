package internal_test

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("ServiceHelper", func() {
	var hdb *appsv1alpha1.HStreamDB

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	It("get service", func() {
		ports := []corev1.ServicePort{
			{
				Name:       "port1",
				Protocol:   corev1.ProtocolTCP,
				Port:       1000,
				TargetPort: intstr.IntOrString{IntVal: 1000},
			},
		}
		compType := appsv1alpha1.ComponentTypeHServer
		svc := internal.GetService(hdb, ports, compType)

		Expect(svc.Name).To(Equal(compType.GetResName(hdb.Name)))
		Expect(svc.Spec.Ports).To(ContainElements(ports[0]))
		Expect(svc.Spec.Selector).To(HaveKeyWithValue(appsv1alpha1.ComponentKey, string(compType)))
	})

	It("get headless service", func() {
		compType := appsv1alpha1.ComponentTypeHServer
		svc := internal.GetHeadlessService(hdb, compType)
		Expect(svc.Name).To(Equal(compType.GetResName(hdb.Name)))
		Expect(svc.Spec.Selector).To(HaveKeyWithValue(appsv1alpha1.ComponentKey, string(compType)))
		Expect(svc.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))

	})
})
