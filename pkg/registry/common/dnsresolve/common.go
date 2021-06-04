// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"fmt"
	"net"
	"net/url"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// DefaultNsmgrProxyService default NSM nsmgr proxy service name for SRV lookup
const DefaultNsmgrProxyService = "nsmgr-proxy.nsm-system"

// DefaultRegistryService default NSM registry service name for SRV lookup
const DefaultRegistryService = "registry.nsm-system"

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

func parseIPPort(domain string) (ip, port interface{}) {
	u, err := url.Parse(domain)
	if err != nil {
		return nil, nil
	}

	ip = net.ParseIP(u.Hostname())
	if ip == nil {
		return nil, nil
	}

	if port = u.Port(); port == "" {
		return nil, nil
	}

	return ip, port
}

func resolveDomain(ctx context.Context, service, domain string, r Resolver) (*url.URL, error) {
	ip, port := parseIPPort(domain)
	if ip == nil || port == nil {
		serviceDomain := fmt.Sprintf("%v.%v", service, domain)

		_, records, err := r.LookupSRV(ctx, "", "", serviceDomain)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, errors.New("resolver.LookupSERV return empty result")
		}
		port = records[0].Port

		ips, err := r.LookupIPAddr(ctx, serviceDomain)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("resolver.LookupIPAddr return empty result")
		}
		ip = ips[0].IP
	}

	u, err := url.Parse(fmt.Sprintf("tcp://%v:%v", ip, port))
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Debug("Resolved url: %v", u)
	return u, nil
}

var _ Resolver = (*net.Resolver)(nil)
