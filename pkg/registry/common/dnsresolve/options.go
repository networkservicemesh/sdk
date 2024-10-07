// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

package dnsresolve

type options struct {
	resolver             Resolver
	nsmgrProxyService    string
	registryService      string
	registryProxyService string
}

// Option is option to configure dnsresovle chain elements
type Option func(*options)

// WithResolver sets DNS resolver by default used net.DefaultResolver
func WithResolver(r Resolver) Option {
	return func(o *options) {
		o.resolver = r
	}
}

// WithNSMgrProxyService sets default service of nsmgr-proxy to lookup DNS SRV records
func WithNSMgrProxyService(service string) Option {
	return func(o *options) {
		o.nsmgrProxyService = service
	}
}

// WithRegistryService sets default service of nsmgr-proxy to lookup DNS SRV records
func WithRegistryService(service string) Option {
	return func(o *options) {
		o.registryService = service
	}
}
