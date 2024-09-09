// Copyright (c) 2022 Cisco and/or its affiliates.
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

package fanout

// Option modifies default fanout dns handler values.
type Option func(*fanoutHandler)

// WithDefaultDNSPort sets default DNS port for fanout dns handler if it is not presented in the client's URL.
func WithDefaultDNSPort(port uint16) Option {
	return func(h *fanoutHandler) {
		h.dnsPort = port
	}
}
