package executor

import (
	"time"
)

var defaultCommandTimeout = 15 * time.Second

type Command struct {
	Command string

	Args []string

	// Timeout specifies the Timeout for running commands.
	Timeout time.Duration
}

func (c *Command) getCommand() []string {
	return append([]string{c.Command}, c.Args...)
}

func (c *Command) getTimeout() time.Duration {
	if c.Timeout != 0 {
		return c.Timeout
	}

	return defaultCommandTimeout
}
