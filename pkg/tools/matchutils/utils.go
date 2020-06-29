// Copyright (c) 2018-2020 VMware, Inc.
//
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
	"bytes"
	"html/template"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// MatchNetworkServices returns true if two network services are matched
func MatchNetworkServices(left, right *registry.NetworkService) bool {
	return (left.Name == "" || left.Name == right.Name) &&
		(left.Payload == "" || left.Payload == right.Payload) &&
		(left.Matches == nil || reflect.DeepEqual(left.Matches, right.Matches))
}

// MatchNetworkServiceEndpoints  returns true if two network service endpoints are matched
func MatchNetworkServiceEndpoints(left, right *registry.NetworkServiceEndpoint) bool {
	return (left.Name == "" || matchString(right.Name, left.Name)) &&
		(left.NetworkServiceLabels == nil || reflect.DeepEqual(left.NetworkServiceLabels, right.NetworkServiceLabels)) &&
		(left.ExpirationTime == nil || left.ExpirationTime.Seconds == right.ExpirationTime.Seconds) &&
		(left.NetworkServiceNames == nil || reflect.DeepEqual(left.NetworkServiceNames, right.NetworkServiceNames)) &&
		(left.Url == "" || matchString(right.Url, left.Url))
}

func matchString(left, right string) bool {
	return strings.Contains(left, right)
}

func isSubset(a, b, nsLabels map[string]string) bool {
	if len(a) < len(b) {
		return false
	}
	for k, v := range b {
		if a[k] != v {
			result := processLabels(v, nsLabels)
			if a[k] != result {
				return false
			}
		}
	}
	return true
}

// SelectEndpoints selects suitable endpoints by labels and network service
func SelectEndpoints(nsLabels map[string]string, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	logrus.Infof("Matching endpoint for labels %v", nsLabels)

	// Iterate through the matches
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !isSubset(nsLabels, match.GetSourceSelector(), nsLabels) {
			continue
		}
		nseCandidates := make([]*registry.NetworkServiceEndpoint, 0)
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range networkServiceEndpoints {
				if isSubset(nse.GetNetworkServiceLabels()[ns.Name].Labels, destination.GetDestinationSelector(), nsLabels) {
					nseCandidates = append(nseCandidates, nse)
				}
			}
		}
		return nseCandidates
	}
	return networkServiceEndpoints
}

// processLabels generates matches based on destination label selectors that specify templating.
func processLabels(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)

	if err != nil {
		panic(err)
	}
	var tmplBytes bytes.Buffer
	err = tmpl.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}
