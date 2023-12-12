package executor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("executor/command", func() {
	command := Command{
		Command: "echo",
		Args:    []string{"hello"},
	}

	It("should get the correct command", func() {
		Expect(command.GetCommand()).To(Equal([]string{"echo", "hello"}))
	})

	It("should get the correct command string", func() {
		Expect(command.ToString()).To(Equal("echo hello"))
	})
})
