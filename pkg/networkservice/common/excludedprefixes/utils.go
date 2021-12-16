// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package excludedprefixes

// RemoveDuplicates - removes duplicate strings from a slice
func RemoveDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	var result []string

	for index := range elements {
		if encountered[elements[index]] {
			continue
		}
		encountered[elements[index]] = true
		result = append(result, elements[index])
	}
	return result
}

// Exclude - excludes strings from a slice
func Exclude(source, exclude []string) []string {
	var prefix string

	for i := 0; i < len(source); i++ {
		prefix = source[i]
		for _, ipCtxPrefix := range exclude {
			if prefix == ipCtxPrefix {
				source = append(source[:i], source[i+1:]...)
				i--
			}
		}
	}

	return source
}
