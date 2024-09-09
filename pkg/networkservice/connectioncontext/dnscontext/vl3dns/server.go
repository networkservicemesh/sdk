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

// Package vl3dns provides a possible for vl3 networkservice endpoint to use distributed dns
package vl3dns

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync/atomic"
	"text/template"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	dnschain "github.com/networkservicemesh/sdk/pkg/tools/dnsutils/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type vl3DNSServer struct {
	dnsServerRecords      genericsync.Map[string, []net.IP]
	dnsConfigs            *genericsync.Map[string, []*networkservice.DNSConfig]
	domainSchemeTemplates []*template.Template
	dnsPort               int
	dnsServer             dnsutils.Handler
	listenAndServeDNS     func(ctx context.Context, handler dnsutils.Handler, listenOn string)
	dnsServerIP           atomic.Value
	dnsServerIPCh         <-chan net.IP
}

type clientDNSNameKey struct{}

// NewServer creates a new vl3dns netwrokservice server.
// It starts dns server on the passed port/url. By default listens ":53".
// By default is using fanout dns handler to connect to other vl3 nses.
// chainCtx is using for signal to stop dns server.
// opts configure vl3dns networkservice instance with specific behavior.
func NewServer(chainCtx context.Context, dnsServerIPCh <-chan net.IP, opts ...Option) networkservice.NetworkServiceServer {
	result := &vl3DNSServer{
		dnsPort:           53,
		listenAndServeDNS: dnsutils.ListenAndServe,
		dnsConfigs:        new(genericsync.Map[string, []*networkservice.DNSConfig]),
		dnsServerIPCh:     dnsServerIPCh,
	}

	for _, opt := range opts {
		opt(result)
	}

	if result.dnsServer == nil {
		result.dnsServer = dnschain.NewDNSHandler(
			dnsconfigs.NewDNSHandler(result.dnsConfigs),
			noloop.NewDNSHandler(),
			norecursion.NewDNSHandler(),
			memory.NewDNSHandler(&result.dnsServerRecords),
			fanout.NewDNSHandler(fanout.WithDefaultDNSPort(uint16(result.dnsPort))),
		)
	}

	result.listenAndServeDNS(chainCtx, result.dnsServer, fmt.Sprintf(":%v", result.dnsPort))

	go func() {
		for {
			select {
			case <-chainCtx.Done():
				return
			case addr, ok := <-dnsServerIPCh:
				if !ok {
					return
				}
				result.dnsServerIP.Store(addr)
			}
		}
	}()

	return result
}

func (n *vl3DNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.Connection.Context.DnsContext = new(networkservice.DNSContext)
	}

	clientsConfigs := request.GetConnection().GetContext().GetDnsContext().GetConfigs()

	recordNames, err := n.buildSrcDNSRecords(request.GetConnection())
	if err != nil {
		return nil, err
	}

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		previousNames := v.([]string)
		if !compareStringSlices(previousNames, recordNames) {
			for _, prevName := range previousNames {
				n.dnsServerRecords.Delete(prevName)
			}
		}
	}

	dnsServerIPStr, err := n.addDNSContext(request.GetConnection(), recordNames)
	if err != nil {
		return nil, err
	}

	resp, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		ips := getSrcIPs(resp)
		if len(ips) > 0 {
			for _, recordName := range recordNames {
				n.dnsServerRecords.Store(recordName, ips)
			}

			metadata.Map(ctx, false).Store(clientDNSNameKey{}, recordNames)
		}
		configs := make([]*networkservice.DNSConfig, 0)
		if srcRoutes := resp.GetContext().GetIpContext().GetSrcRoutes(); len(srcRoutes) > 0 {
			lastPrefix := srcRoutes[len(srcRoutes)-1].Prefix
			for _, config := range clientsConfigs {
				for _, serverIP := range config.GetDnsServerIps() {
					if dnsServerIPStr == serverIP {
						continue
					}
					if withinPrefix(serverIP, lastPrefix) {
						configs = append(configs, config)
					}
				}
			}
		}
		n.dnsConfigs.Store(resp.GetId(), configs)
	}
	return resp, err
}

func (n *vl3DNSServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	n.dnsConfigs.Delete(conn.GetId())

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		names := v.([]string)
		for _, name := range names {
			n.dnsServerRecords.Delete(name)
		}
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (n *vl3DNSServer) addDNSContext(c *networkservice.Connection, dnsRecords []string) (serverIP string, err error) {
	if ip := n.dnsServerIP.Load(); ip != nil {
		dnsServerIP := ip.(net.IP)

		// Construct searchDomains for a client
		// Example: dnsRecord = "target.d1.d2.d3." ---> searchDomain = ["d1.d2.d3", "d2.d3", "d3"]
		var searchDomains []string
		for _, dnsRecord := range dnsRecords {
			var ok bool
			searchDomain := dnsRecord
			for _, searchDomain, ok = strings.Cut(strings.Trim(searchDomain, "."), "."); ok; _, searchDomain, ok = strings.Cut(searchDomain, ".") {
				searchDomains = append(searchDomains, searchDomain)
			}
		}

		// Add dnsConfig to the connection
		dnsContext := c.GetContext().GetDnsContext()
		configToAdd := &networkservice.DNSConfig{
			DnsServerIps:  []string{dnsServerIP.String()},
			SearchDomains: searchDomains,
		}
		if !dnsutils.ContainsDNSConfig(dnsContext.GetConfigs(), configToAdd) {
			dnsContext.Configs = append(dnsContext.Configs, configToAdd)
		}
		return dnsServerIP.String(), nil
	} else if c.GetPath().GetPathSegments()[0].GetName() == c.GetCurrentPathSegment().GetName() {
		// If it calls itself - this is not an error, but a request to allocate a dns address
		return "", nil
	}
	return "", errors.New("DNS address is initializing")
}

func (n *vl3DNSServer) buildSrcDNSRecords(c *networkservice.Connection) ([]string, error) {
	var result []string
	for _, templ := range n.domainSchemeTemplates {
		recordBuilder := new(strings.Builder)
		if err := templ.Execute(recordBuilder, c); err != nil {
			return nil, errors.Wrap(err, "error occurred executing the template or writing its output")
		}
		result = append(result, removeDupDots(recordBuilder.String()))
	}
	return result, nil
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func withinPrefix(ipAddr, prefix string) bool {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false
	}
	pool := ippool.NewWithNet(ipNet)
	return pool.ContainsString(ipAddr)
}

func getSrcIPs(c *networkservice.Connection) []net.IP {
	var ips []net.IP
	for _, srcIPNet := range c.GetContext().GetIpContext().GetSrcIPNets() {
		ips = append(ips, srcIPNet.IP)
	}
	return ips
}

var regexDot = regexp.MustCompile(`\.+`)

func removeDupDots(str string) string {
	return regexDot.ReplaceAllString(str, ".")
}
