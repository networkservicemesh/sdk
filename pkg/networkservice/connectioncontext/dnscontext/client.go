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
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	modifiedResolvConfKey = "# modified"
)

type dnsContextClient struct {
	chainContext        context.Context
	resolveConfigPath   string
	defaultNameServerIP string
	resolvconfDNSConfig *networkservice.DNSConfig
	dnsConfigsMap       *dnsconfig.Map
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

	if !dnsutils.ContainsDNSConfig(request.GetConnection().GetContext().GetDnsContext().Configs, c.resolvconfDNSConfig) {
		request.GetConnection().GetContext().GetDnsContext().Configs = append(request.GetConnection().GetContext().GetDnsContext().Configs, c.resolvconfDNSConfig)
	}

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
	bytes, err := os.ReadFile(c.resolveConfigPath)
	modifiedResolvConf := string(bytes)
	if err != nil || modifiedResolvConf == "" {
		return
	}

	if !strings.HasPrefix(modifiedResolvConf, modifiedResolvConfKey) {
		return
	}

	originalResolvConf := make([]string, 0)
	for _, line := range strings.Split(modifiedResolvConf, "\n")[1:] {
		if strings.HasPrefix(line, "#") {
			originalResolvConf = append(originalResolvConf, line[1:])
		}
	}

	_ = os.WriteFile(c.resolveConfigPath, []byte(strings.Join(originalResolvConf, "\n")), os.ModePerm)
}

func (c *dnsContextClient) storeOriginalResolvConf() {
	bytes, err := os.ReadFile(c.resolveConfigPath)
	originalResolvConfig := string(bytes)
	if err != nil {
		return
	}

	lines := strings.Split(originalResolvConfig, "\n")
	modifiedResolvConf := make([]string, len(lines)+1)
	modifiedResolvConf[0] = modifiedResolvConfKey

	for i, line := range lines {
		modifiedResolvConf[i+1] = "#" + line
	}

	_ = os.WriteFile(c.resolveConfigPath, []byte(strings.Join(modifiedResolvConf, "\n")+"\n\n"), os.ModePerm)
}

func (c *dnsContextClient) appendResolvConf(resolvConf string) error {
	resolvConfFile, err := os.OpenFile(c.resolveConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	if _, err := resolvConfFile.Write([]byte(resolvConf)); err != nil {
		resolvConfFile.Close()
		return err
	}

	return resolvConfFile.Close()
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

	if err = c.appendResolvConf(r.Serialize()); err != nil {
		log.FromContext(c.chainContext).Errorf("An error during appending resolve config: %v", err.Error())
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
