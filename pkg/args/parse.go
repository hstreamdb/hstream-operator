/*
Copyright 2023 HStream Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package args

import (
	"strings"
)

func ParseArgs(args []string) map[string]string {
	argsMap := make(map[string]string)

	var currentKey string

	for _, arg := range args {
		// Check if the argument has the format --key
		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			currentKey = arg
			argsMap[currentKey] = ""
		} else if currentKey != "" {
			// If a key is set, treat the argument as its value
			argsMap[currentKey] = arg
			currentKey = ""
		}
	}

	return argsMap
}
