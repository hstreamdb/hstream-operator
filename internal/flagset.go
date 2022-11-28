package internal

import (
	"fmt"
	"sort"
)

type FlagSet struct {
	actual   map[string]string
	args     []string // arguments after flags
	lastName string
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	f.args = arguments

	if f.actual == nil {
		f.actual = make(map[string]string)
	}

	for {
		seen, err := f.parseOne()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		return err
	}
	return nil
}

// parseOne parses one flag. It reports whether a flag was seen.
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}

	s := f.args[0]
	if len(s) != 0 {
		if s[0] == '-' {
			if f.lastName != "" {
				return false, fmt.Errorf("bad flag syntax: %s", s)
			}

			numMinuses := 1
			if s[1] == '-' {
				numMinuses++
				if len(s) == 2 { // "--" terminates the flags
					f.args = f.args[1:]
					return false, nil
				}
			}
			// it's a flag, value is the next arg
			f.lastName = s[numMinuses:]
			f.actual[f.lastName] = ""
		} else {
			if f.lastName == "" {
				return false, fmt.Errorf("bad flag syntax: %s", s)
			}

			// it's a value
			value := s
			if value[0] == '-' || value[0] == '=' {
				return false, fmt.Errorf("bad flag syntax: %s", s)
			}
			f.actual[f.lastName] = value
			// reset lastName, next arg is flag
			f.lastName = ""
		}
	} else {
		// reset lastName, next arg is flag
		f.lastName = ""
	}

	if len(f.args) > 0 {
		f.args = f.args[1:]
		return true, nil
	}
	return false, nil
}

// Visit visits the flags in lexicographical order, calling fn for each.
func (f *FlagSet) Visit(fn func(flag, value string)) {
	for _, flag := range sortFlags(f.actual) {
		fn(flag, f.actual[flag])
	}
}

// Flags returns parsed flags
func (f *FlagSet) Flags() map[string]string { return f.actual }

// sortFlags returns the flags as a slice in lexicographical sorted order.
func sortFlags(flags map[string]string) []string {
	result := make([]string, len(flags))
	i := 0
	for name := range flags {
		result[i] = name
		i++
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
