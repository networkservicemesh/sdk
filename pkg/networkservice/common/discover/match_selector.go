// Copyright (c) 2018-2022 VMware, Inc.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package discover

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

func matchEndpoint(clockTime clock.Clock, nsLabels map[string]string, ns *registry.NetworkService, nses ...*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	validNetworkServiceEndpoints := validateExpirationTime(clockTime, nses)
	// Iterate through the matches
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !matchutils.IsSubset(nsLabels, match.GetSourceSelector(), nsLabels) {
			continue
		}
		nseCandidates := make([]*registry.NetworkServiceEndpoint, 0)
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range validNetworkServiceEndpoints {
				var candidateNetworkServiceLabels = nse.GetNetworkServiceLabels()[ns.GetName()]
				var labels map[string]string
				if candidateNetworkServiceLabels != nil {
					labels = candidateNetworkServiceLabels.Labels
				}
				if matchutils.IsSubset(labels, destination.GetDestinationSelector(), nsLabels) {
					nseCandidates = append(nseCandidates, nse)
				}
			}
		}

		if match.Fallthrough && len(nseCandidates) == 0 {
			continue
		}

		if match.GetMetadata() != nil && len(match.Routes) == 0 && len(nseCandidates) == 0 {
			break
		}

		return nseCandidates
	}

	return validNetworkServiceEndpoints
}

func validateExpirationTime(clockTime clock.Clock, nses []*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	var validNetworkServiceEndpoints []*registry.NetworkServiceEndpoint
	for _, nse := range nses {
		if nse.GetExpirationTime() == nil || nse.GetExpirationTime().AsTime().After(clockTime.Now()) {
			validNetworkServiceEndpoints = append(validNetworkServiceEndpoints, nse)
		}
	}

	return validNetworkServiceEndpoints
}
