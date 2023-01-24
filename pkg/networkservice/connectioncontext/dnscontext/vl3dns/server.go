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
	"strings"
	"sync"
	"text/template"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	dnschain "github.com/networkservicemesh/sdk/pkg/tools/dnsutils/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vl3DNSServer struct {
	chanCtx               context.Context
	dnsServerRecords      memory.Map
	dnsConfigs            *dnsconfig.Map
	domainSchemeTemplates []*template.Template
	dnsPort               int
	dnsServer             dnsutils.Handler
	listenAndServeDNS     func(ctx context.Context, handler dnsutils.Handler, listenOn string)
	dnsServerIP           net.IP
	serverAddressCh       <-chan net.IP
	executor              serialize.Executor
	once                  sync.Once
}

type clientDNSNameKey struct{}

// NewServer creates a new vl3dns netwrokservice server.
// It starts dns server on the passed port/url. By default listens ":53".
// By default is using fanout dns handler to connect to other vl3 nses.
// chanCtx is using for signal to stop dns server.
// opts confugre vl3dns networkservice instance with specific behavior.
func NewServer(chanCtx context.Context, serverAddressCh <-chan net.IP, opts ...Option) networkservice.NetworkServiceServer {
	var result = &vl3DNSServer{
		chanCtx:           chanCtx,
		dnsPort:           53,
		listenAndServeDNS: dnsutils.ListenAndServe,
		dnsConfigs:        new(dnsconfig.Map),
		serverAddressCh:   serverAddressCh,
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

	result.listenAndServeDNS(chanCtx, result.dnsServer, fmt.Sprintf(":%v", result.dnsPort))

	if len(serverAddressCh) > 0 {
		result.dnsServerIP = <-serverAddressCh
	}
	return result
}

func (n *vl3DNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	n.once.Do(func() {
		go n.checkServerAddressUpdates(ctx)
	})

	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.Connection.Context.DnsContext = new(networkservice.DNSContext)
	}

	var clientsConfigs = request.GetConnection().GetContext().GetDnsContext().GetConfigs()

	dnsServerIPStr, added := n.addDNSContext(request.GetConnection())
	var recordNames, err = n.buildSrcDNSRecords(request.GetConnection())

	if err != nil {
		return nil, err
	}

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		var previousNames = v.([]string)
		if !compareStringSlices(previousNames, recordNames) {
			for _, prevName := range previousNames {
				n.dnsServerRecords.Delete(prevName)
			}
		}
	}

	ips := getSrcIPs(request.GetConnection())
	if len(ips) > 0 {
		<-n.executor.AsyncExec(func() {
			n.storeSrcIPs(ctx, recordNames, ips)
		})
	}

	resp, err := next.Server(ctx).Request(ctx, request)

	if err == nil {
		configs := make([]*networkservice.DNSConfig, 0)
		if srcRoutes := resp.GetContext().GetIpContext().GetSrcRoutes(); len(srcRoutes) > 0 {
			var lastPrefix = srcRoutes[len(srcRoutes)-1].Prefix
			for _, config := range clientsConfigs {
				for _, serverIP := range config.DnsServerIps {
					if added && dnsServerIPStr == serverIP {
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
	n.dnsConfigs.Delete(conn.Id)

	if v, ok := metadata.Map(ctx, false).LoadAndDelete(clientDNSNameKey{}); ok {
		var names = v.([]string)
		for _, name := range names {
			n.dnsServerRecords.Delete(name)
		}
	}

	return next.Server(ctx).Close(ctx, conn)
}

func (n *vl3DNSServer) addDNSContext(c *networkservice.Connection) (added string, ok bool) {
	if dnsServerIP := n.dnsServerIP; dnsServerIP != nil {
		var dnsContext = c.GetContext().GetDnsContext()
		configToAdd := &networkservice.DNSConfig{
			DnsServerIps: []string{dnsServerIP.String()},
		}
		if !dnsutils.ContainsDNSConfig(dnsContext.Configs, configToAdd) {
			dnsContext.Configs = append(dnsContext.Configs, configToAdd)
		}
		return dnsServerIP.String(), true
	}
	return "", false
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

func (n *vl3DNSServer) checkServerAddressUpdates(ctx context.Context) {
	for {
		select {
		case <-n.chanCtx.Done():
			return
		case addr, ok := <-n.serverAddressCh:
			if !ok {
				return
			}

			n.updateServerAddress(ctx, addr)
		}
	}
}

func (n *vl3DNSServer) updateServerAddress(ctx context.Context, address net.IP) {
	<-n.executor.AsyncExec(func() {
		n.dnsServerIP = address
		logger := log.FromContext(ctx).WithField("vl3DNSServer", "Request")

		if eventConsumer, ok := monitor.LoadEventConsumer(ctx, metadata.IsClient(n)); ok {
			_ = eventConsumer.Send(&networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: eventConsumer.GetConnections(),
			})
		} else {
			logger.Debug("eventConsumer is not presented")
		}
	})
}

func (n *vl3DNSServer) storeSrcIPs(ctx context.Context, recordNames []string, ips []net.IP) {
	for _, recordName := range recordNames {
		n.dnsServerRecords.Store(recordName, ips)
	}

	metadata.Map(ctx, false).Store(clientDNSNameKey{}, recordNames)
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
	var pool = ippool.NewWithNet(ipNet)
	return pool.ContainsString(ipAddr)
}

func getSrcIPs(c *networkservice.Connection) []net.IP {
	var ips []net.IP
	for _, srcIPNet := range c.GetContext().GetIpContext().GetSrcIPNets() {
		ips = append(ips, srcIPNet.IP)
	}
	return ips
}
