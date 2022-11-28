package internal_test

import (
	"github.com/hstreamdb/hstream-operator/internal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flagset", func() {
	var flag *internal.FlagSet

	BeforeEach(func() {
		flag = &internal.FlagSet{}
	})

	Context("valid args", func() {
		var args []string
		var err error
		BeforeEach(func() {
			args = []string{
				"--e", "/config.json",
				"--b", "$(POD_IP)",
				"--d", "$(POD_NAME)",
				"--c", "/etc/logdevice",
				"--a", "1",
			}
			err = flag.Parse(args)
		})
		It("should successfully parse", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("get the sorted flag", func() {
			var sorted []string
			flag.Visit(func(flag, _ string) {
				sorted = append(sorted, flag)
			})
			Expect(sorted).To(Equal([]string{"a", "b", "c", "d", "e"}))
		})
		It("get the actual flags", func() {
			actual := flag.Flags()
			Expect(actual).To(HaveLen(5))
			Expect(actual).Should(BeComparableTo(map[string]string{
				"a": "1",
				"b": "$(POD_IP)",
				"c": "/etc/logdevice",
				"d": "$(POD_NAME)",
				"e": "/config.json",
			}))
		})
	})
	Context("arg only has flag", func() {
		var args []string
		var err error
		It("should successfully parse 1", func() {
			args = []string{
				"--a", "",
				"--b", "1",
				"--c", "",
			}
			err = flag.Parse(args)
			Expect(err).ToNot(HaveOccurred())

			actual := flag.Flags()
			Expect(actual).To(BeComparableTo(map[string]string{
				"a": "",
				"b": "1",
				"c": "",
			}))
		})
		It("should successfully parse 2", func() {
			args = []string{
				"--b", "1",
				"--a", "",
				"--c", "",
			}
			err = flag.Parse(args)
			Expect(err).ToNot(HaveOccurred())

			actual := flag.Flags()
			Expect(actual).To(BeComparableTo(map[string]string{
				"a": "",
				"b": "1",
				"c": "",
			}))
		})
		It("should successfully parse 3", func() {
			args = []string{
				"--a",
			}
			err = flag.Parse(args)
			Expect(err).ToNot(HaveOccurred())

			actual := flag.Flags()
			Expect(actual).To(HaveKeyWithValue("a", ""))
		})
		It("should successfully parse 4", func() {
			args = []string{
				"--a", "1",
			}
			err = flag.Parse(args)
			Expect(err).ToNot(HaveOccurred())

			actual := flag.Flags()
			Expect(actual).To(HaveKeyWithValue("a", "1"))
		})
	})
	Context("invalid args", func() {
		var args []string
		var err error
		It("flag isn't begin with '--' or '-'", func() {
			args = []string{
				"--e", "/config.json",
				"b", "$(POD_IP)",
			}
			err = flag.Parse(args)
			Expect(err).To(HaveOccurred())
		})
		It("value is begin with '--'", func() {
			// flag must have a value though it is empty string
			args = []string{
				"--e", "--a",
			}
			err = flag.Parse(args)
			Expect(err).To(HaveOccurred())
		})
		It("value is begin with '-'", func() {
			// flag must have a value though it is empty string
			args = []string{
				"--e", "-a",
			}
			err = flag.Parse(args)
			Expect(err).To(HaveOccurred())
		})
	})
})
