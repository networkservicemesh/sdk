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

package vl3dns

import (
	"context"
	"net"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vl3DNSClient struct {
	dnsServerIP net.IP
	dnsConfigs  *genericsync.Map[string, []*networkservice.DNSConfig]
}

// NewClient - returns a new null client that does nothing but call next.Client(ctx).{Request/Close} and return the result
//
//	This is very useful in testing
func NewClient(dnsServerIP net.IP, dnsConfigs *genericsync.Map[string, []*networkservice.DNSConfig]) networkservice.NetworkServiceClient {
	return &vl3DNSClient{
		dnsServerIP: dnsServerIP,
		dnsConfigs:  dnsConfigs,
	}
}

func (n *vl3DNSClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		request.Connection = new(networkservice.Connection)
	}
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = new(networkservice.ConnectionContext)
	}
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.GetConnection().GetContext().DnsContext = new(networkservice.DNSContext)
	}

	request.GetConnection().GetContext().GetDnsContext().Configs = []*networkservice.DNSConfig{
		{
			DnsServerIps: []string{n.dnsServerIP.String()},
		},
	}
	resp, err := next.Client(ctx).Request(ctx, request, opts...)

	if err == nil {
		configs := make([]*networkservice.DNSConfig, 0)
		for _, config := range resp.GetContext().GetDnsContext().GetConfigs() {
			skip := false
			for _, ip := range config.GetDnsServerIps() {
				if ip == n.dnsServerIP.String() {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			configs = append(configs, config)
		}

		n.dnsConfigs.Store(resp.GetId(), configs)
	}

	return resp, err
}

func (n *vl3DNSClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	n.dnsConfigs.Delete(conn.GetId())
	return next.Client(ctx).Close(ctx, conn, opts...)
}
