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

func removeDuplicates(elements []string) []string {
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

func exclude(source, exclude []string) []string {
	var s string
	excludeMap := make(map[string]struct{})

	for _, e := range exclude {
		excludeMap[e] = struct{}{}
	}

	for i := 0; i < len(source); i++ {
		s = source[i]
		if _, ok := excludeMap[s]; ok {
			source = append(source[:i], source[i+1:]...)
			i--
		}
	}

	return source
}
