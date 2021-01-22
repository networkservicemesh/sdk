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

package dnscontext

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/networkservicemesh/sdk/pkg/tools/logger/logruslogger"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnscontext"
)

type dnsContextClient struct {
	cancelMonitoring    context.CancelFunc
	chainContext        context.Context
	monitorContext      context.Context
	requestContext      context.Context
	coreFilePath        string
	resolveConfigPath   string
	defaultNameServerIP string
	monitorCallOptions  []grpc.CallOption
	dnsConfigManager    dnscontext.Manager
	monitorClient       networkservice.MonitorConnectionClient
	executor            serialize.Executor
}

// NewClient creates a new DNS client chain component. Setups all DNS traffic to the localhost. Monitors DNS configs from connections.
func NewClient(monitorClient networkservice.MonitorConnectionClient, options ...DNSOption) networkservice.NetworkServiceClient {
	c := &dnsContextClient{
		chainContext:        context.Background(),
		dnsConfigManager:    dnscontext.NewManager(),
		monitorClient:       monitorClient,
		defaultNameServerIP: "127.0.0.1",
		resolveConfigPath:   "/etc/resolv.conf",
		coreFilePath:        "/etc/coredns/Corefile",
	}
	for _, o := range options {
		o.apply(c)
	}
	_, log := logruslogger.New(context.Background())
	if r, err := dnscontext.OpenResolveConfig(c.resolveConfigPath); err != nil {
		log.Errorf("DnsContextClient: can not load resolve config file. Path: %v. Error: %v", c.resolveConfigPath, err.Error())
	} else {
		c.dnsConfigManager.Store("", &networkservice.DNSConfig{
			SearchDomains: r.Value(dnscontext.AnyDomain),
			DnsServerIps:  r.Value(dnscontext.NameserverProperty),
		})
		r.SetValue(dnscontext.NameserverProperty, c.defaultNameServerIP)
		if err := r.Save(); err != nil {
			log.Errorf("DnsContextClient: can not update resolve config file. Error: %v", err.Error())
		}
	}
	return c
}

func (c *dnsContextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	c.requestContext = ctx
	c.executor.AsyncExec(c.monitorConfigs)
	return rv, err
}

func (c *dnsContextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	r, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	c.cancelMonitoring()
	return r, err
}

func (c *dnsContextClient) monitorConfigs() {
	c.monitorContext, c.cancelMonitoring = context.WithCancel(c.chainContext)
	steam, err := c.monitorClient.MonitorConnections(c.monitorContext, &networkservice.MonitorScopeSelector{}, c.monitorCallOptions...)
	if err != nil {
		logger.Log(c.requestContext).Errorf("DnsContextClient: Can not start monitor connections: %v", err)
		c.executor.AsyncExec(c.monitorConfigs)
		return
	}
	for {
		if c.monitorContext.Err() != nil {
			return
		}
		event, err := steam.Recv()
		if err != nil {
			c.executor.AsyncExec(c.monitorConfigs)
			return
		}
		c.handleEvent(event)
		v := c.dnsConfigManager.String()
		logger.Log(c.requestContext).Info(v)
		_ = ioutil.WriteFile(c.coreFilePath, []byte(v), os.ModePerm)
	}
}

func (c *dnsContextClient) handleEvent(event *networkservice.ConnectionEvent) {
	switch event.GetType() {
	case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
		for k, v := range event.Connections {
			c.dnsConfigManager.Store(k, v.GetContext().GetDnsContext().GetConfigs()...)
		}
	case networkservice.ConnectionEventType_DELETE:
		for k := range event.Connections {
			c.dnsConfigManager.Remove(k)
		}
	}
}
