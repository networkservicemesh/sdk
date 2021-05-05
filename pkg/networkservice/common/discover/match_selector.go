// Copyright (c) 2018-2020 VMware, Inc.
//
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

package discover

import (
	"bytes"
	"text/template"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

// isSubset checks if B is a subset of A. TODO: reconsider this as a part of "tools"
func isSubset(a, b, nsLabels map[string]string) bool {
	if len(a) < len(b) {
		return false
	}
	for k, v := range b {
		if a[k] != v {
			result := ProcessLabels(v, nsLabels)
			if a[k] != result {
				return false
			}
		}
	}
	return true
}

func filterValidNSEs(clockTime clock.Clock, nses ...*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	var validNetworkServiceEndpoints []*registry.NetworkServiceEndpoint
	for _, nse := range nses {
		if nse.GetExpirationTime() == nil || nse.GetExpirationTime().AsTime().After(clockTime.Now()) {
			validNetworkServiceEndpoints = append(validNetworkServiceEndpoints, nse)
		}
	}
	return validNetworkServiceEndpoints
}

func matchEndpoint(nsLabels map[string]string, ns *registry.NetworkService, nses []*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	var match *registry.Match
	// Iterate through the matches
	for _, m := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !isSubset(nsLabels, m.GetSourceSelector(), nsLabels) {
			continue
		}
		match = m
		break
	}

	if match == nil {
		return nses
	}

	var nseCandidates []*registry.NetworkServiceEndpoint
	// Check all Destinations in that match
	for _, destination := range match.GetRoutes() {
		// Each NSE should be matched against that destination
		for _, nse := range nses {
			if isSubset(nse.GetNetworkServiceLabels()[ns.Name].Labels, destination.GetDestinationSelector(), nsLabels) {
				nseCandidates = append(nseCandidates, nse)
			}
		}
	}
	return nseCandidates
}

// ProcessLabels generates matches based on destination label selectors that specify templating.
func ProcessLabels(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)

	if err != nil {
		panic(err)
	}
	return process(tmpl, vars)
}

func process(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}
