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

package dnscontext

import (
	"context"
	"os"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestServerBasic(t *testing.T) {
	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "0",
			NetworkService: "icmp",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					DstIpAddr: "172.16.1.2",
				},
			},
		},
	}
	s := next.NewNetworkServiceServer(NewServer(ConfigFromConnection()))
	resp, err := s.Request(context.Background(), r)
	require.Nil(t, err)
	require.NotNil(t, resp.GetContext().GetDnsContext())
	require.Len(t, resp.Context.DnsContext.Configs[0].SearchDomains, 1)
	require.Len(t, resp.Context.DnsContext.Configs[0].DnsServerIps, 1)
	require.Equal(t, resp.Context.DnsContext.Configs[0].SearchDomains[0], r.Connection.NetworkService)
	require.Equal(t, resp.Context.DnsContext.Configs[0].DnsServerIps[0], r.Connection.Context.IpContext.DstIpAddr)
}

func TestServerBasicFromEnv(t *testing.T) {
	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}
	expectedDomain := "test.domain"
	expectedIP := "8.8.8.8"
	err := os.Setenv("DNSCONFIG_DNSSERVERIPS", expectedIP)
	require.Nil(t, err)
	err = os.Setenv("DNSCONFIG_SEARCHDOMAINS", expectedDomain)
	require.Nil(t, err)
	s := next.NewNetworkServiceServer(NewServer(ConfigFromEnv()))
	resp, err := s.Request(context.Background(), r)
	require.Nil(t, err)
	require.NotNil(t, resp.GetContext().GetDnsContext())
	require.Len(t, resp.Context.DnsContext.Configs[0].SearchDomains, 1)
	require.Len(t, resp.Context.DnsContext.Configs[0].DnsServerIps, 1)
	require.Equal(t, resp.Context.DnsContext.Configs[0].SearchDomains[0], expectedDomain)
	require.Equal(t, resp.Context.DnsContext.Configs[0].DnsServerIps[0], expectedIP)
}
