// Copyright (c) 2019-2020 VMware, Inc.
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

// Package roundrobin provides a networkservice chain element that round robins among the candidates for providing
// a requested networkservice
package roundrobin

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/algorithm/roundrobin"
)

type roundRobinSelector struct {
	nsRoundRobins roundRobinMap
}

func (s *roundRobinSelector) selectEndpoint(ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	if s == nil || len(networkServiceEndpoints) == 0 {
		return nil
	}

	rr, _ := s.nsRoundRobins.LoadOrStore(ns.GetName(), &roundrobin.RoundRobin{})
	idx := rr.Index(len(networkServiceEndpoints))

	return networkServiceEndpoints[idx]
}
