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

// Package vl3dns provides a possible for vl3 networkservice endpoint to use distributed dns
package vl3dns

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"text/template"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	dnsnext "github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

type vl3DNSServer struct {
	dnsServerRecords      memory.Map
	fanoutAddresses       sync.Map
	domainSchemeTemplates []*template.Template
	initialFanoutList     []url.URL
	dnsPort               int
	dnsServer             dnsutils.Handler
	listenAndServeDNS     func(ctx context.Context, handler dnsutils.Handler, listenOn string)
}

type clientDNSNameKey struct{}

// NewServer creates a new vl3dns netwrokservice server.
// It starts dns server on the passed port/url. By default listens ":53".
// By default is using fanout dns handler to connect to other vl3 nses.
// chanCtx is using for signal to stop dns server.
// opts confugre vl3dns networkservice instance with specific behavior.
func NewServer(chanCtx context.Context, opts ...Option) networkservice.NetworkServiceServer {
	var result = &vl3DNSServer{
		dnsPort:           53,
		listenAndServeDNS: dnsutils.ListenAndServe,
	}

	for _, opt := range opts {
		opt(result)
	}

	if result.dnsServer == nil {
		result.dnsServer = dnsnext.NewDNSHandler(
			noloop.NewDNSHandler(),
			norecursion.NewDNSHandler(),
			memory.NewDNSHandler(&result.dnsServerRecords),
			fanout.NewDNSHandler(result.getFanoutAddresses),
		)
	}

	result.listenAndServeDNS(chanCtx, result.dnsServer, fmt.Sprintf(":%v", result.dnsPort))

	return result
}

func (n *vl3DNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.Connection.Context.DnsContext = new(networkservice.DNSContext)
	}

	var ipContext = request.GetConnection().GetContext().GetIpContext()

	for _, dstIPNet := range ipContext.GetDstIPNets() {
		request.GetConnection().GetContext().GetDnsContext().Configs = append(request.GetConnection().GetContext().GetDnsContext().Configs, &networkservice.DNSConfig{
			DnsServerIps: []string{dstIPNet.IP.String()},
		})
	}

	var recordNames, err = n.buildSrcDNSRecords(request.GetConnection())

	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, srcIPNet := range request.GetConnection().GetContext().GetIpContext().GetSrcIPNets() {
		ips = append(ips, srcIPNet.IP)
	}

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		var previousNames = v.([]string)
		if !compareStringSlices(previousNames, recordNames) {
			for _, prevName := range previousNames {
				n.dnsServerRecords.Delete(prevName)
			}
		}
	}
	if len(ips) > 0 {
		for _, recordName := range recordNames {
			n.dnsServerRecords.Store(recordName, ips)
		}

		metadata.Map(ctx, false).Store(clientDNSNameKey{}, recordNames)
	}

	if n.shouldAddToFanoutList(ipContext) {
		for _, srcIPNet := range ipContext.GetSrcIPNets() {
			var u = url.URL{Scheme: "tcp", Host: fmt.Sprintf("%v:%v", srcIPNet.IP.String(), n.dnsPort)}
			n.fanoutAddresses.Store(u, struct{}{})
		}
	}

	return next.Server(ctx).Request(ctx, request)
}

func (n *vl3DNSServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	for _, srcIPNet := range conn.Context.GetIpContext().GetSrcIPNets() {
		var u = url.URL{Scheme: "tcp", Host: fmt.Sprintf("%v:%v", srcIPNet.IP.String(), n.dnsPort)}
		n.fanoutAddresses.Delete(u)
	}

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		var names = v.([]string)
		for _, name := range names {
			n.dnsServerRecords.Delete(name)
		}
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (n *vl3DNSServer) buildSrcDNSRecords(c *networkservice.Connection) ([]string, error) {
	var result []string
	for _, templ := range n.domainSchemeTemplates {
		var recordBuilder = new(strings.Builder)
		if err := templ.Execute(recordBuilder, c); err != nil {
			return nil, err
		}
		result = append(result, recordBuilder.String())
	}
	return result, nil
}

func (n *vl3DNSServer) getFanoutAddresses() []url.URL {
	var result = n.initialFanoutList
	n.fanoutAddresses.Range(func(key, _ interface{}) bool {
		result = append(result, key.(url.URL))
		return true
	})
	return result
}

func (n *vl3DNSServer) shouldAddToFanoutList(ipContext *networkservice.IPContext) bool {
	if len(ipContext.SrcRoutes) > 0 {
		var lastSrcRoute = ipContext.SrcRoutes[len(ipContext.SrcRoutes)-1]
		_, ipNet, err := net.ParseCIDR(lastSrcRoute.Prefix)
		if err != nil {
			return false
		}
		var pool = ippool.NewWithNet(ipNet)
		logrus.Error(ipNet.String())
		for _, srcIP := range ipContext.GetSrcIpAddrs() {
			if !pool.ContainsNetString(srcIP) {
				return true
			}
		}
	}
	return false
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
