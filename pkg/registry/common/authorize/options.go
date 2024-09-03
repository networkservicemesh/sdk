// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type options struct {
	policies           policiesList
	resourcePathIdsMap *genericsync.Map[string, []string]
}

// Option is authorization option for server.
type Option func(*options)

// Any authorizes any call of request/close.
func Any() Option {
	return func(o *options) {
		o.policies = nil
	}
}

// WithPolicies sets custom policies for registry.
// policyPaths can be combination of both policy files and dirs with policies.
func WithPolicies(policyPaths ...string) Option {
	return func(o *options) {
		policies, err := opa.PoliciesByFileMask(policyPaths...)
		if err != nil {
			panic(errors.Wrap(err, "failed to read policies in NetworkServiceRegistry authorize client").Error())
		}

		for _, p := range policies {
			o.policies = append(o.policies, Policy(p))
		}
	}
}

// WithResourcePathIdsMap sets map to keep resourcePathIdsMap to authorize connections with Registry Authorize Chain Element.
func WithResourcePathIdsMap(m *genericsync.Map[string, []string]) Option {
	return func(o *options) {
		o.resourcePathIdsMap = m
	}
}
