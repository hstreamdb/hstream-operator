package internal_test

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PodSetHelper", func() {
	var hdb *hapi.HStreamDB
	compType := hapi.ComponentTypeHServer
	comp := &hapi.Component{
		Replicas: 1,
	}
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"label": "testLabel",
			},
		},
		Spec: corev1.PodSpec{},
	}

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	It("get deployment", func() {
		deploy := internal.GetDeployment(hdb, comp, podSpec, compType)
		Expect(deploy.Name).To(Equal(compType.GetResName(hdb)))
		Expect(deploy.Spec.Selector.MatchLabels).To(HaveKeyWithValue("label", "testLabel"))
		Expect(*deploy.Spec.Replicas).To(Equal(comp.Replicas))
		Expect(deploy.Annotations).To(HaveKey(hapi.LastSpecKey))
		Expect(&deploy.Spec.Template).To(Equal(podSpec))
	})

	It("get statefulSet", func() {
		sts := internal.GetStatefulSet(hdb, comp, podSpec, compType)
		service := internal.GetHeadlessService(hdb, compType)

		Expect(sts.Name).To(Equal(compType.GetResName(hdb)))
		Expect(sts.Spec.Selector.MatchLabels).To(HaveKeyWithValue("label", "testLabel"))
		Expect(*sts.Spec.Replicas).To(Equal(comp.Replicas))
		Expect(sts.Annotations).To(HaveKey(hapi.LastSpecKey))
		Expect(&sts.Spec.Template).To(Equal(podSpec))
		Expect(sts.Spec.ServiceName).To(Equal(service.Name))
		Expect(sts.Spec.PodManagementPolicy).To(Equal(appsv1.ParallelPodManagement))
	})

	It("get object hash", func() {
		type object struct {
			Name   string
			Events []string
			Labels map[string]string
		}
		obj1 := object{
			Name:   "obj",
			Events: []string{"event1", "event2"},
			Labels: map[string]string{
				"l1": "l1",
				"l2": "l2",
			},
		}
		obj2 := object{
			Name:   "obj",
			Events: []string{"event2", "event1"},
			Labels: map[string]string{
				"l2": "l2",
				"l1": "l1",
			},
		}
		h1 := internal.GetObjectHash(&obj1)
		h2 := internal.GetObjectHash(&obj2)
		Expect(h1).To(Equal(h2))
	})
})
