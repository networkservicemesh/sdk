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
	"reflect"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// MatchNetworkServices returns true if two network services are matched
func MatchNetworkServices(left, right *registry.NetworkService) bool {
	return (left.Name == "" || matchString(right.Name, left.Name)) &&
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
