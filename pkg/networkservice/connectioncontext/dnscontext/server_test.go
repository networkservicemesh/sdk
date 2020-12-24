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

package dnscontext_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestServerBasic(t *testing.T) {
	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}
	configs := func() []*networkservice.DNSConfig {
		return []*networkservice.DNSConfig{
			{
				DnsServerIps:  []string{"8.8.8.8"},
				SearchDomains: []string{"sample1"},
			},
			{
				DnsServerIps:  []string{"8.8.4.4"},
				SearchDomains: []string{"sample2"},
			},
		}
	}
	expected := configs()
	s := next.NewNetworkServiceServer(dnscontext.NewServer(configs))
	resp, err := s.Request(context.Background(), r)
	require.Nil(t, err)
	require.NotNil(t, resp.GetContext().GetDnsContext())
	require.Equal(t, resp.GetContext().GetDnsContext().Configs, expected)
	r.Connection.Context.DnsContext = nil
	_, err = s.Close(context.Background(), r.Connection)
	require.Nil(t, err)
	require.Equal(t, resp.GetContext().GetDnsContext().Configs, expected)
}

func TestServerShouldNotPanicIfPassNil(t *testing.T) {
	require.NotPanics(t, func() {
		ctx := logger.WithLog(context.Background())
		_, _ = next.NewNetworkServiceServer(dnscontext.NewServer(nil)).Request(ctx, nil)
	})
}
