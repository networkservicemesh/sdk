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

package dnscontext_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_DNSContextClient_Restart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")
	err := ioutil.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\n"), os.ModePerm)
	require.NoError(t, err)
	const expectedEmptyCorefile = `. {
	fanout . 8.8.4.4
	log
	reload
	cache {
		denial 0
	}
}`
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var c = chain.NewNetworkServiceClient(
			metadata.NewClient(),
			dnscontext.NewClient(
				dnscontext.WithCorefilePath(corefilePath),
				dnscontext.WithResolveConfigPath(resolveConfigPath),
				dnscontext.WithChainContext(ctx),
			),
		)
		_, _ = c.Request(ctx, &networkservice.NetworkServiceRequest{})

		cancel()
	}

	require.Never(t, func() bool {
		for {
			// #nosec
			b, err := ioutil.ReadFile(corefilePath)
			if err == nil {
				time.Sleep(time.Millisecond * 50)
				continue
			}
			return string(b) != expectedEmptyCorefile
		}
	}, time.Second/2, time.Millisecond*100)
}

func Test_DNSContextClient_Usecases(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err := ioutil.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\nsearch example.com\n"), os.ModePerm)
	require.NoError(t, err)

	client := chain.NewNetworkServiceClient(
		dnscontext.NewClient(
			dnscontext.WithCorefilePath(corefilePath),
			dnscontext.WithResolveConfigPath(resolveConfigPath),
			dnscontext.WithChainContext(ctx),
		),
	)

	const expectedEmptyCorefile = `. {
	fanout . 8.8.4.4
	log
	reload
	cache {
		denial 0
	}
}`

	requireFileChanged(ctx, t, corefilePath, expectedEmptyCorefile)

	var samples = []struct {
		request          *networkservice.NetworkServiceRequest
		expectedCorefile string
	}{
		{
			expectedCorefile: ". {\n\tfanout . 8.8.4.4\n\tlog\n\treload\n\tcache {\n\t\tdenial 0\n\t}\n}\nexample.com {\n\tfanout . 8.8.8.8\n\tlog\n\tcache {\n\t\tdenial 0\n\t}\n}",
			request: &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "nsc-1",
					Context: &networkservice.ConnectionContext{
						DnsContext: &networkservice.DNSContext{
							Configs: []*networkservice.DNSConfig{
								{
									SearchDomains: []string{"example.com"},
									DnsServerIps:  []string{"8.8.8.8"},
								},
							},
						},
					},
				},
			},
		},
		{
			expectedCorefile: ". {\n\tfanout . 8.8.4.4\n\tlog\n\treload\n\tcache {\n\t\tdenial 0\n\t}\n}\nexample.com {\n\tfanout . 7.7.7.7 8.8.8.8 9.9.9.9\n\tlog\n\tcache {\n\t\tdenial 0\n\t}\n}",
			request: &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Id: "nsc-1",
					Context: &networkservice.ConnectionContext{
						DnsContext: &networkservice.DNSContext{
							Configs: []*networkservice.DNSConfig{
								{
									SearchDomains: []string{"example.com"},
									DnsServerIps:  []string{"7.7.7.7"},
								},
								{
									SearchDomains: []string{"example.com"},
									DnsServerIps:  []string{"8.8.8.8"},
								},
								{
									SearchDomains: []string{"example.com"},
									DnsServerIps:  []string{"9.9.9.9"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, s := range samples {
		resp, err := client.Request(ctx, s.request)
		require.NoError(t, err)
		require.NotNil(t, resp.GetContext().GetDnsContext())
		require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), len(s.request.GetConnection().Context.DnsContext.GetConfigs()))
		requireFileChanged(ctx, t, corefilePath, s.expectedCorefile)
		_, err = client.Close(ctx, resp)
		require.NoError(t, err)

		requireFileChanged(ctx, t, corefilePath, expectedEmptyCorefile)
	}
}

func requireFileChanged(ctx context.Context, t *testing.T, location, expected string) {
	var r string
	for ctx.Err() == nil {
		b, err := ioutil.ReadFile(filepath.Clean(location))
		r = string(b)
		if err == nil && r == expected {
			return
		}
		runtime.Gosched()
	}
	require.FailNowf(t, "fail to wait update", "file has not updated. Last content: %s, expected: %s", r, expected)
}
