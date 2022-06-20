// Copyright (c) 2022 Cisco and/or its affiliates.
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

package dnscontext

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	dnsAddressesKey  contextKeyType = "DNSAddresses"
	searchDomainsKey contextKeyType = "SearchDomains"
)

type contextKeyType string

func WithDNSAddresses(parent context.Context, addresses []url.URL) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	log.FromContext(parent).Debugf("passed DNS addresses: %v", addresses)
	return context.WithValue(parent, dnsAddressesKey, addresses)
}

func DNSAddresses(ctx context.Context) []url.URL {
	if rv, ok := ctx.Value(dnsAddressesKey).([]url.URL); ok {
		return rv
	}
	return nil
}

func WithSearchDomains(parent context.Context, domains []string) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	log.FromContext(parent).Debugf("passed search domains: %v", domains)
	return context.WithValue(parent, searchDomainsKey, domains)
}

func SearchDomains(ctx context.Context) []string {
	if rv, ok := ctx.Value(searchDomainsKey).([]string); ok {
		return rv
	}
	return nil
}
