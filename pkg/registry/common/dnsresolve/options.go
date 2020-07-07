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

package dnsresolve

type configurable interface {
	setResolver(r Resolver)
	setDomain(domain *Domain)
}

// Option is option to configure dnsresovle chain elements
type Option interface {
	apply(configurable)
}

type optionApplyFunc func(configurable)

func (f optionApplyFunc) apply(c configurable) {
	f(c)
}

// WithDefaultDomain sets default domain that is used if a domain was not passed via context
func WithDefaultDomain(domain Domain) Option {
	return optionApplyFunc(func(c configurable) {
		c.setDomain(&domain)
	})
}

// WithResolver sets DNS resolver by default used net.DefaultResolver
func WithResolver(r Resolver) Option {
	return optionApplyFunc(func(c configurable) {
		c.setResolver(r)
	})
}
