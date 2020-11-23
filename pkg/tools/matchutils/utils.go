// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package matchutils provides utils to match network services and network service endpoints
package matchutils

import (
	"strings"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// MatchNetworkServices returns true if two network services are matched
func MatchNetworkServices(left, right *registry.NetworkService) bool {
	return (left.Name == "" || strings.Contains(right.Name, left.Name)) &&
		(left.Payload == "" || left.Payload == right.Payload) &&
		(left.Matches == nil || cmp.Equal(left.Matches, right.Matches, cmp.Comparer(proto.Equal)))
}

// MatchNetworkServiceEndpoints  returns true if two network service endpoints are matched
func MatchNetworkServiceEndpoints(left, right *registry.NetworkServiceEndpoint) bool {
	return (left.Name == "" || strings.Contains(right.Name, left.Name)) &&
		(left.NetworkServiceLabels == nil || labelsContains(right.NetworkServiceLabels, left.NetworkServiceLabels)) &&
		(left.ExpirationTime == nil || left.ExpirationTime.Seconds == right.ExpirationTime.Seconds) &&
		(left.NetworkServiceNames == nil || contains(right.NetworkServiceNames, left.NetworkServiceNames)) &&
		(left.Url == "" || strings.Contains(right.Url, left.Url))
}

func labelsContains(where, what map[string]*registry.NetworkServiceLabels) bool {
	for lService, lLabels := range what {
		rService, ok := where[lService]
		if !ok {
			return false
		}
		for lKey, lVal := range lLabels.Labels {
			rVal, ok := rService.Labels[lKey]
			if !ok || lVal != rVal {
				return false
			}
		}
	}
	return true
}

func contains(where, what []string) bool {
	set := make(map[string]struct{})
	for _, s := range what {
		set[s] = struct{}{}
	}
	for _, s := range where {
		delete(set, s)
	}
	return len(set) == 0
}
