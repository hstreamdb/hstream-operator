package executor

import (
	"strings"
)

type Command struct {
	Command string

	Args []string

	// Timeout specifies the Timeout for running commands.
	// Timeout time.Duration
}

func (c *Command) GetCommand() []string {
	return append([]string{c.Command}, c.Args...)
}

func (c *Command) ToString() string {
	return c.Command + " " + strings.Join(c.Args, " ")
}
