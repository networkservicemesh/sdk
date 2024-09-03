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

package dnscontext_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func Test_DNSContextServer_Request(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}
	expected := []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{"8.8.8.8"},
			SearchDomains: []string{"sample1"},
		},
		{
			DnsServerIps:  []string{"8.8.4.4"},
			SearchDomains: []string{"sample2"},
		},
	}

	s := next.NewNetworkServiceServer(dnscontext.NewServer(&networkservice.DNSConfig{
		DnsServerIps:  []string{"8.8.8.8"},
		SearchDomains: []string{"sample1"},
	}, &networkservice.DNSConfig{
		DnsServerIps:  []string{"8.8.4.4"},
		SearchDomains: []string{"sample2"},
	}))
	resp, err := s.Request(ctx, r)
	require.NoError(t, err)
	require.NotNil(t, resp.GetContext().GetDnsContext())
	require.Equal(t, resp.GetContext().GetDnsContext().GetConfigs(), expected)
}

func Test_DNSContextServer_Close(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := dnscontext.NewServer().Close(ctx, new(networkservice.Connection))
	require.NoError(t, err)
}
