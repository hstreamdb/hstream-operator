package internal_test

import (
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PodHelper", func() {
	var hdb *hapi.HStreamDB

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	Context("with base nil", func() {
		It("get default label and namespace", func() {
			meta := internal.GetObjectMetadata(hdb, nil, hapi.ComponentTypeHServer)
			Expect(meta.Labels).NotTo(BeNil())
			Expect(meta.Annotations).NotTo(BeNil())
			Expect(meta.Labels).To(HaveKeyWithValue(hapi.InstanceKey, hdb.Name))
			Expect(meta.Labels).To(HaveKeyWithValue(hapi.ComponentKey, string(hapi.ComponentTypeHServer)))
			Expect(meta.Namespace).To(Equal(hdb.Namespace))
		})
	})
	Context("with base not nil", func() {
		It("copy label and annotation from base", func() {
			base := &metav1.ObjectMeta{
				Labels: map[string]string{
					"label": "testLabel",
				},
				Annotations: map[string]string{
					"annotation": "testAnnotation",
				},
			}
			meta := internal.GetObjectMetadata(hdb, base, hapi.ComponentTypeHServer)
			Expect(meta.Labels).To(HaveKeyWithValue(hapi.InstanceKey, hdb.Name))
			Expect(meta.Labels).To(HaveKeyWithValue(hapi.ComponentKey, string(hapi.ComponentTypeHServer)))
			Expect(meta.Labels).To(HaveKeyWithValue("label", "testLabel"))
			Expect(meta.Annotations).To(HaveKeyWithValue("annotation", "testAnnotation"))
			Expect(meta.Namespace).To(Equal(hdb.Namespace))

		})
	})
})
