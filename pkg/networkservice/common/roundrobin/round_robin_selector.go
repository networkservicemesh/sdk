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
	"sync"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type roundRobinSelector struct {
	//nolint:nolintlint //TODO - replace this with something like a sync.Map that doesn't require us to lock all requests for a simple
	// selection
	sync.Mutex
	roundRobin map[string]int
}

func newRoundRobinSelector() *roundRobinSelector {
	return &roundRobinSelector{
		roundRobin: make(map[string]int),
	}
}

func (rr *roundRobinSelector) selectEndpoint(ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	if rr == nil || len(networkServiceEndpoints) == 0 {
		return nil
	}
	rr.Lock()
	defer rr.Unlock()
	idx := rr.roundRobin[ns.GetName()] % len(networkServiceEndpoints)
	endpoint := networkServiceEndpoints[idx]
	if endpoint == nil {
		return nil
	}
	rr.roundRobin[ns.GetName()] = rr.roundRobin[ns.GetName()] + 1
	return endpoint
}
