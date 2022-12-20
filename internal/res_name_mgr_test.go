package internal_test

import (
	appsv1alpha1 "github.com/hstreamdb/hstream-operator/api/v1alpha1"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResNameMgr", func() {
	var hdb *appsv1alpha1.HStreamDB

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
	})

	Context("with shortName not empty", func() {
		It("get res name with shortName", func() {
			name := internal.GetResNameOnPanic(hdb, "test")
			Expect(name).To(Equal(hdb.Name + "-" + "test"))
		})
	})

	Context("with default value", func() {
		It("get res name with default name", func() {
			name := internal.GetResNameWithDefault(hdb, "", "default_name")
			Expect(name).To(Equal(hdb.Name + "-" + "default_name"))
		})

		It("get res name with shortName", func() {
			name := internal.GetResNameWithDefault(hdb, "test", "default_name")
			Expect(name).To(Equal(hdb.Name + "-" + "test"))
		})
	})
})
