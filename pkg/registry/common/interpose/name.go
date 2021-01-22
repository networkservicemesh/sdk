// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package interpose

import "strings"

// namePrefix - a common prefix for all registered cross NSEs
const namePrefix = "interpose-nse#"

func interposeName(name string) string {
	return namePrefix + name
}

// Is returns true if passed name contains interpose identity
func Is(name string) bool {
	return strings.HasPrefix(name, namePrefix)
}
