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

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

// Resolver is DNS resolver
type Resolver interface {
	// LookupSRV tries to resolve an SRV query of the given service,
	// protocol, and domain name. The proto is "tcp" or "udp".
	// The returned records are sorted by priority and randomized
	// by weight within a priority.
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
	// LookupIPAddr looks up host using the local resolver.
	// It returns a slice of that host's IPv4 and IPv6 addresses.
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func resolveDomain(ctx context.Context, service, domain string, r Resolver) (*url.URL, error) {
	_, records, err := r.LookupSRV(ctx, service, "tcp", domain)

	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, errors.New("resolver.LookupSERV return empty result")
	}

	ips, err := r.LookupIPAddr(ctx, records[0].Target)

	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("resolver.LookupIPAddr return empty result")
	}

	u, err := url.Parse(fmt.Sprintf("tcp://%v:%v", ips[0].IP, records[0].Port))

	if err != nil {
		return nil, err
	}

	return u, nil
}

var _ Resolver = (*net.Resolver)(nil)
