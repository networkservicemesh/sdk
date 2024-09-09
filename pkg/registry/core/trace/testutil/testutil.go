// Copyright (c) 2023 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testutil has few util functions for testing
package testutil

import (
	"fmt"
	"strings"
)

// TrimLogTime - to format logs.
func TrimLogTime(buff fmt.Stringer) string {
	// Logger created by the trace chain element uses custom formatter, which prints date and time info in each line
	// To check if output matches our expectations, we need to somehow get rid of this info.
	// We have the following options:
	// 1. Configure formatter options on logger creation in trace element
	// 2. Use some global configuration (either set global default formatter
	// 	  instead of creating it in trace element or use global config for our formatter)
	// 3. Remove datetime information from the output
	// Since we are unlikely to need to remove date in any case except these tests,
	// it seems like the third option would be the most convenient.
	result := ""
	datetimeLength := 19
	for _, line := range strings.Split(buff.String(), "\n") {
		if len(line) > datetimeLength {
			result += line[datetimeLength:] + "\n"
		} else {
			result += line
		}
	}

	return result
}
