// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// DefaultNsmgrProxyService default NSM nsmgr proxy service name for SRV lookup.
const DefaultNsmgrProxyService = "nsmgr-proxy.nsm-system"

// DefaultRegistryService default NSM registry service name for SRV lookup.
const DefaultRegistryService = "registry.nsm-system"

// Resolver is DNS resolver.
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

func parseIPPort(domain string) (ip net.IP, port string) {
	u, err := url.Parse(domain)
	if err != nil {
		return nil, ""
	}

	ip = net.ParseIP(u.Hostname())
	if ip == nil {
		return nil, ""
	}

	if port = u.Port(); port == "" {
		return nil, ""
	}

	return ip, port
}

func resolveDomain(ctx context.Context, service, domain string, r Resolver) (u *url.URL, err error) {
	defer func() {
		if err == nil {
			log.FromContext(ctx).Debugf("Resolved url: %v", u)
		}
	}()

	ip, port := parseIPPort(domain)
	if ip == nil || port == "" {
		serviceDomain := fmt.Sprintf("%v.%v", service, domain)

		_, records, err := r.LookupSRV(ctx, "", "", serviceDomain)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve a SRV query for a %s", serviceDomain)
		}
		if len(records) == 0 {
			return nil, errors.New("resolver.LookupSERV return empty result")
		}
		port = strconv.Itoa(int(records[0].Port))

		ips, err := r.LookupIPAddr(ctx, serviceDomain)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to look up a host using a local resolver for a %s", serviceDomain)
		}
		if len(ips) == 0 {
			return nil, errors.New("resolver.LookupIPAddr return empty result")
		}
		ip = ips[0].IP
	}

	return formatURL(ip.String(), port)
}

// formatURL formats ip and port into a TCP URL. It accounts for IPV4
// and IPV6 addresses. If error is non-nil then the URL pointer is
// undefined.
func formatURL(ip, port string) (*url.URL, error) {
	// Assume the address is IPV4
	urlStr := fmt.Sprintf("tcp://%s:%s", ip, port)

	// If ip is IPV6 then wrap it in brackets.
	if strings.Count(ip, ":") >= 2 {
		urlStr = fmt.Sprintf("tcp://[%s]:%s", ip, port)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url %s", urlStr)
	}
	return parsedURL, nil
}

var _ Resolver = (*net.Resolver)(nil)
