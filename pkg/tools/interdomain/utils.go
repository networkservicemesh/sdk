// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

package interdomain

import (
	"strings"
)

const identifier = "@"

// Is returns true if passed string can be represented as interdomain URL
func Is(s string) bool {
	return strings.Contains(s, identifier)
}

// Not returns true if passed string is not related to interdomain.
func Not(s string) bool {
	return !Is(s)
}

// Join concatenates strings with intedomain identifier
func Join(s ...string) string {
	return strings.Join(s, identifier)
}

// Domain returns domain name from interdomain query
func Domain(s string) string {
	pieces := strings.SplitN(s, identifier, 2)
	if len(pieces) < 2 {
		return ""
	}
	return pieces[1]
}

// Target returns target name from interdomain query
func Target(s string) string {
	pieces := strings.SplitN(s, identifier, 2)
	return pieces[0]
}
