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

package externaldnscontext_test

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/retry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/externaldnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func newTestingDNSRegisterServcer(ctx context.Context, t *testing.T) *testingDNSRegisterService {
	var s = grpc.NewServer()

	var r = new(testingDNSRegisterService)
	dns.RegisterDNSServer(s, r)

	var serverAddr url.URL

	require.Len(t, grpcutils.ListenAndServe(ctx, &serverAddr, s), 0)

	r.Addr = serverAddr
	return r
}

type testingDNSRegisterService struct {
	entries sync.Map
	Addr    url.URL
}

func (e *testingDNSRegisterService) lenEntries() int {
	var r int

	e.entries.Range(func(key, value interface{}) bool {
		r++
		return true
	})

	return r
}

func (e *testingDNSRegisterService) FetchConfigs(ctx context.Context, _ *empty.Empty) (*dns.Configs, error) {
	return &dns.Configs{Configs: []*networkservice.DNSConfig{{DnsServerIps: []string{"8.8.8.8"}}}}, nil
}

func (e *testingDNSRegisterService) ManageNames(s dns.DNS_ManageNamesServer) error {
	for s.Context().Err() == nil {
		r, err := s.Recv()
		if err != nil {
			return err
		}
		var values []string

		for _, v := range r.Labels {
			values = append(values, v)
		}

		sort.Strings(values)

		var name = strings.Join(values, ".")

		if r.Type == dns.Type_ASSIGN {
			e.entries.Store(name, r.Ips[0])
		} else {
			e.entries.Delete(name)
		}

		err = s.Send(&dns.DNSResponse{
			Names: []string{name},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func Test_ExternalDNSContext_EmptyRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var s = newTestingDNSRegisterServcer(ctx, t)

	var c = retry.NewClient(
		adapters.NewServerToClient(
			externaldnscontext.NewServer(
				ctx,
				map[string]string{"app": "vl3", "podName": "nse1"},
				&s.Addr, grpc.WithInsecure(),
			),
		),
		retry.WithTryTimeout(time.Second/10),
		retry.WithInterval(time.Millisecond*100),
	)

	_, err := c.Request(ctx, &networkservice.NetworkServiceRequest{})

	require.NoError(t, err)
}

func Test_ExternalDNSContext_RequestRefreshClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var s = newTestingDNSRegisterServcer(ctx, t)

	var c = retry.NewClient(
		adapters.NewServerToClient(
			externaldnscontext.NewServer(
				ctx,
				map[string]string{"app": "vl3", "podName": "nse1"},
				&s.Addr, grpc.WithInsecure(),
			),
		),
		retry.WithTryTimeout(time.Second/10),
		retry.WithInterval(time.Millisecond*100),
	)

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: t.Name(),
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"1.1.1.1/32"},
					DstIpAddrs: []string{"1.1.1.2/32"},
				},
			},
		},
	}

	_, err := c.Request(ctx, req)

	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return s.lenEntries() == 2
	}, time.Second/2, time.Second/10)

	resp, err := c.Request(ctx, req)

	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return s.lenEntries() == 2
	}, time.Second/2, time.Second/10)

	_, err = c.Close(ctx, resp)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return s.lenEntries() == 0
	}, time.Second/2, time.Second/10)
}
func Test_ExternalDNSContext_WorksCorrectly_WithDNSContext(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var s = newTestingDNSRegisterServcer(ctx, t)

	var c = retry.NewClient(
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				externaldnscontext.NewServer(
					ctx,
					map[string]string{"app": "vl3", "podName": "nse1"},
					&s.Addr, grpc.WithInsecure(),
				),
				dnscontext.NewServer(&networkservice.DNSConfig{DnsServerIps: []string{"8.8.4.4"}}),
			),
		),
		retry.WithTryTimeout(time.Second/10),
		retry.WithInterval(time.Millisecond*100),
	)

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: t.Name(),
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"1.1.1.1/32"},
					DstIpAddrs: []string{"1.1.1.2/32"},
				},
			},
		},
	}

	resp, err := c.Request(ctx, req)

	require.NoError(t, err)

	var ips []string

	for _, cfg := range resp.GetContext().GetDnsContext().GetConfigs() {
		ips = append(ips, cfg.DnsServerIps...)
	}

	require.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, ips)
}
