// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"bytes"
	"strings"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// MatchEndpoint filters input nses by network service configuration.
// Returns the same nses list if no matches are declared in the network service.
func MatchEndpoint(nsLabels map[string]string, ns *registry.NetworkService, nses ...*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	if len(ns.GetMatches()) == 0 {
		return nses
	}
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !IsSubset(nsLabels, match.GetSourceSelector(), nsLabels) {
			continue
		}
		nseCandidates := make([]*registry.NetworkServiceEndpoint, 0)
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range nses {
				candidateNetworkServiceLabels := nse.GetNetworkServiceLabels()[ns.GetName()]
				var labels map[string]string
				if candidateNetworkServiceLabels != nil {
					labels = candidateNetworkServiceLabels.GetLabels()
				}
				if IsSubset(labels, destination.GetDestinationSelector(), nsLabels) {
					nseCandidates = append(nseCandidates, nse)
				}
			}
		}

		if match.GetFallthrough() && len(nseCandidates) == 0 {
			continue
		}

		if match.GetMetadata() != nil && len(match.GetRoutes()) == 0 && len(nseCandidates) == 0 {
			return nses
		}

		return nseCandidates
	}

	return nil
}

// MatchNetworkServices returns true if two network services are matched.
func MatchNetworkServices(left, right *registry.NetworkService) bool {
	return (left.GetName() == "" || right.GetName() == left.GetName()) &&
		(left.GetPayload() == "" || left.GetPayload() == right.GetPayload()) &&
		(left.Matches == nil || cmp.Equal(left.GetMatches(), right.GetMatches(), cmp.Comparer(proto.Equal)))
}

// MatchNetworkServiceEndpoints  returns true if two network service endpoints are matched.
func MatchNetworkServiceEndpoints(left, right *registry.NetworkServiceEndpoint) bool {
	return (left.GetName() == "" || right.GetName() == left.GetName()) &&
		(left.NetworkServiceLabels == nil || labelsContains(right.GetNetworkServiceLabels(), left.GetNetworkServiceLabels())) &&
		(left.GetExpirationTime() == nil || left.GetExpirationTime().GetSeconds() == right.GetExpirationTime().GetSeconds()) &&
		(left.NetworkServiceNames == nil || contains(right.GetNetworkServiceNames(), left.GetNetworkServiceNames())) &&
		(left.GetUrl() == "" || strings.Contains(right.GetUrl(), left.GetUrl()))
}

// IsSubset checks if B is a subset of A.
// Tries to process values for each B value.
func IsSubset(a, b, values map[string]string) bool {
	if len(a) < len(b) {
		return false
	}
	for k, v := range b {
		if a[k] != v {
			result := processLabels(v, values)
			if a[k] != result {
				return false
			}
		}
	}
	return true
}

// processLabels generates matches based on destination label selectors that specify templating.
func processLabels(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)
	if err != nil {
		return str
	}

	rv, err := process(tmpl, vars)
	if err != nil {
		return str
	}

	return rv
}

func process(t *template.Template, vars interface{}) (string, error) {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		return "", errors.Wrap(err, "error during execution of template")
	}
	return tmplBytes.String(), nil
}

func labelsContains(where, what map[string]*registry.NetworkServiceLabels) bool {
	for lService, lLabels := range what {
		rService, ok := where[lService]
		if !ok {
			return false
		}
		for lKey, lVal := range lLabels.GetLabels() {
			rVal, ok := rService.GetLabels()[lKey]
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
