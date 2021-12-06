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

package vl3ipam

import (
	"net"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

func exclude(prefixes ...string) (ipv4exclude, ipv6exclude *ippool.IPPool) {
	ipv4exclude = ippool.New(net.IPv4len)
	ipv6exclude = ippool.New(net.IPv6len)
	for _, prefix := range prefixes {
		ipv4exclude.AddNetString(prefix)
		ipv6exclude.AddNetString(prefix)
	}
	return
}
