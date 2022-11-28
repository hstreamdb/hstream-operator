package v1alpha1_test

import (
	"github.com/hstreamdb/hstream-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComponentType", func() {
	var compType v1alpha1.ComponentType
	Context("Component type is StatefulSet", func() {
		It("hstore should be StatefulSet", func() {
			compType = v1alpha1.ComponentTypeHStore
			Expect(compType.IsStateful()).To(Equal(true))
		})

		It("hstore should be StatefulSet", func() {
			compType = v1alpha1.ComponentTypeHServer
			Expect(compType.IsStateful()).To(Equal(true))
		})
	})

	Context("Component type isn't StatefulSet", func() {
		It("admin server should not be StatefulSet", func() {
			compType = v1alpha1.ComponentTypeAdminServer
			Expect(compType.IsStateful()).To(Equal(false))
		})
	})
})
