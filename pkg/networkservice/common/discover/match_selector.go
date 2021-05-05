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

func matchEndpoint(nsLabels map[string]string, nsName string, match *registry.Match, nses []*registry.NetworkServiceEndpoint) ([]*registry.NetworkServiceEndpoint, []map[string]string) {
	if match == nil {
		return nses, nil
	}

	var nseCandidates []*registry.NetworkServiceEndpoint
	var destLabels []map[string]string
	for _, destination := range match.GetRoutes() {
		for _, nse := range nses {
			if isSubset(nse.GetNetworkServiceLabels()[nsName].Labels, destination.GetDestinationSelector(), nsLabels) {
				nseCandidates = append(nseCandidates, nse)
				destLabels = append(destLabels, destination.GetDestinationSelector())
			}
		}
	}
	return nseCandidates, destLabels
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
