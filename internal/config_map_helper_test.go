package internal_test

import (
	"github.com/hstreamdb/hstream-operator/internal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigMapHelper", func() {
	Context("with config map set visiting", func() {
		It("get sorted config map", func() {
			var mapNames []string
			internal.ConfigMaps.Visit(func(m internal.ConfigMap) {
				mapNames = append(mapNames, m.MountName)
			})
			Expect(mapNames).To(Equal([]string{internal.LogDeviceConfig, internal.NShardsConfig}))
		})
		It("should get specify config map", func() {
			cm, has := internal.ConfigMaps.Get(internal.LogDeviceConfig)
			Expect(has).To(BeTrue())
			Expect(cm.MountName).To(Equal(internal.LogDeviceConfig))
		})
		It("should not get config map", func() {
			_, has := internal.ConfigMaps.Get("not exist")
			Expect(has).To(BeFalse())
		})
	})

	Context("with log device config parsing", func() {
		It("should parse successfully", func() {
			raw := []byte("{}")
			Expect(internal.ParseLogDeviceConfig(raw)).NotTo(BeNil())
		})
		It("should get error", func() {
			raw := []byte("abc:123")
			_, err := internal.ParseLogDeviceConfig(raw)
			Expect(err).To(HaveOccurred())
		})
		It("should not return error", func() {
			var raw []byte
			Expect(internal.ParseLogDeviceConfig(raw)).To(HaveLen(0))
		})
	})
})
