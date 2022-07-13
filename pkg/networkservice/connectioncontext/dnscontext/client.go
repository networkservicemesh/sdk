// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	dnsContextClientRefreshKey = "dnsContextClientRefreshKey"
)

type dnsContextClient struct {
	chainContext           context.Context
	resolveConfigPath      string
	storedResolvConfigPath string
	defaultNameServerIP    string
	resolvconfDNSConfig    *networkservice.DNSConfig
	dnsConfigsMap          *dnsconfig.Map
}

// NewClient creates a new DNS client chain component. Setups all DNS traffic to the localhost. Monitors DNS configs from connections.
func NewClient(options ...DNSOption) networkservice.NetworkServiceClient {
	var c = &dnsContextClient{
		chainContext:        context.Background(),
		defaultNameServerIP: "127.0.0.1",
		resolveConfigPath:   "/etc/resolv.conf",
	}
	for _, o := range options {
		o.apply(c)
	}

	c.storedResolvConfigPath = c.resolveConfigPath + ".restore"
	c.initialize()

	return c
}

func (c *dnsContextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}
	if request.Connection.GetContext() == nil {
		request.Connection.Context = &networkservice.ConnectionContext{
			DnsContext: &networkservice.DNSContext{},
		}
	}
	if request.Connection.GetContext().GetDnsContext() == nil {
		request.Connection.Context.DnsContext = &networkservice.DNSContext{}
	}

	initialClientDNSConfigs, _ := metadata.Map(ctx, true).LoadOrStore(dnsContextClientRefreshKey, request.Connection.Context.DnsContext.Configs)
	request.Connection.Context.DnsContext.Configs = append(initialClientDNSConfigs.([]*networkservice.DNSConfig), c.resolvconfDNSConfig)

	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	var configs []*networkservice.DNSConfig
	if rv.GetContext().GetDnsContext() != nil {
		configs = rv.GetContext().GetDnsContext().GetConfigs()
	}

	c.dnsConfigsMap.Store(rv.Id, configs)

	return rv, err
}

func (c *dnsContextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.dnsConfigsMap.Delete(conn.Id)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (c *dnsContextClient) restoreResolvConf() {
	originalResolvConf, err := ioutil.ReadFile(c.storedResolvConfigPath)
	if err != nil || len(originalResolvConf) == 0 {
		return
	}
	_ = os.WriteFile(c.resolveConfigPath, originalResolvConf, os.ModePerm)
}

func (c *dnsContextClient) storeOriginalResolvConf() {
	if _, err := os.Stat(c.storedResolvConfigPath); err == nil {
		return
	}
	originalResolvConf, err := ioutil.ReadFile(c.resolveConfigPath)
	if err != nil {
		return
	}
	_ = ioutil.WriteFile(c.storedResolvConfigPath, originalResolvConf, os.ModePerm)
}

func (c *dnsContextClient) initialize() {
	c.restoreResolvConf()

	r, err := openResolveConfig(c.resolveConfigPath)
	if err != nil {
		log.FromContext(c.chainContext).Errorf("An error during open resolve config: %v", err.Error())
		return
	}

	c.storeOriginalResolvConf()

	nameserver := r.Value(nameserverProperty)
	if !containsNameserver(nameserver, c.defaultNameServerIP) {
		c.resolvconfDNSConfig = &networkservice.DNSConfig{
			SearchDomains: r.Value(searchProperty),
			DnsServerIps:  nameserver,
		}
	}

	r.SetValue(nameserverProperty, c.defaultNameServerIP)
	r.SetValue(searchProperty, []string{}...)

	if err = r.Save(); err != nil {
		log.FromContext(c.chainContext).Errorf("An error during save resolve config: %v", err.Error())
		return
	}
}

func containsNameserver(servers []string, value string) bool {
	for i := range servers {
		if servers[i] == value {
			return true
		}
	}

	return false
}
