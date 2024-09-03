// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2023 Cisco Systems, Inc.
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

package authorize

import (
	"github.com/edwarnicke/genericsync"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type options struct {
	policyPaths           []string
	spiffeIDConnectionMap *genericsync.Map[spiffeid.ID, *genericsync.Map[string, struct{}]]
}

// Option is authorization option for network service server.
type Option func(*options)

// Any authorizes any call of request/close.
func Any() Option {
	return WithPolicies([]string{}...)
}

// WithPolicies sets custom policies for networkservice.
// policyPaths can be combination of both policy files and dirs with policies.
func WithPolicies(policyPaths ...string) Option {
	return func(o *options) {
		o.policyPaths = policyPaths
	}
}

// WithSpiffeIDConnectionMap sets map to keep spiffeIDConnectionMap to authorize connections with MonitorConnectionServer.
func WithSpiffeIDConnectionMap(s *genericsync.Map[spiffeid.ID, *genericsync.Map[string, struct{}]]) Option {
	return func(o *options) {
		o.spiffeIDConnectionMap = s
	}
}
