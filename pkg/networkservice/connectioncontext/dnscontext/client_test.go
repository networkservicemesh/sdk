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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
)

const (
	resolvConf = `nameserver 8.8.4.4
nameserver 6.6.3.3
search example.com
`
	expectedResolvConf = `nameserver 127.0.0.1
nameserver 8.8.4.4
nameserver 6.6.3.3
search example.com
`
	expectedEmptyCoreFile = `. {
	log
	reload
}
`
	expectedEmptyCoreFileWithServer = `. {
	forward . %s
	log
	reload
}
`
)

func TestDNSContextClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var samples = []struct {
		name                string
		defaultNameServerIP net.IP
		configs             []*networkservice.DNSConfig
		expectedCorefile    string
	}{
		{
			name: "without default, without conflicts",
			configs: []*networkservice.DNSConfig{
				{
					SearchDomains: []string{"example.com"},
					DnsServerIps:  []string{"8.8.8.8"},
				},
			},
			expectedCorefile: ". {\n\tlog\n\treload\n}\nexample.com {\n\tforward . 8.8.8.8\n\tlog\n}\n",
		},
		{
			name:                "with default, without conflicts",
			defaultNameServerIP: net.IPv4(10, 10, 10, 10),
			configs: []*networkservice.DNSConfig{
				{
					SearchDomains: []string{"example.com"},
					DnsServerIps:  []string{"8.8.8.8"},
				},
			},
			expectedCorefile: ". {\n\tforward . 10.10.10.10\n\tlog\n\treload\n}\nexample.com {\n\tforward . 8.8.8.8\n\tlog\n}\n",
		},
		{
			name: "without default, with conflicts",
			configs: []*networkservice.DNSConfig{
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
			expectedCorefile: ". {\n\tlog\n\treload\n}\nexample.com {\n\tfanout . 7.7.7.7 8.8.8.8 9.9.9.9\n\tlog\n}\n",
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testDNSContextClient(ctx, t, sample.defaultNameServerIP, sample.configs, sample.expectedCorefile)
		})
	}
}

func testDNSContextClient(
	ctx context.Context,
	t *testing.T,
	defaultNameServerIP net.IP,
	configs []*networkservice.DNSConfig,
	expectedCoreFile string,
) {
	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err := ioutil.WriteFile(resolveConfigPath, []byte(resolvConf), os.ModePerm)
	require.NoError(t, err)

	// 0. `cmd-nsc-init` case
	// 1. `cmd-nsc` case
	for i := 0; i < 2; i++ {
		opts := []dnscontext.DNSOption{
			dnscontext.WithChainContext(ctx),
			dnscontext.WithCorefilePath(corefilePath),
			dnscontext.WithResolveConfigPath(resolveConfigPath),
		}
		if defaultNameServerIP != nil {
			opts = append(opts, dnscontext.WithDefaultNameServerIP(defaultNameServerIP))
		}

		client := dnscontext.NewClient(opts...)

		requireFileChanged(ctx, t, resolveConfigPath, expectedResolvConf)

		emptyCoreFile := expectedEmptyCoreFile
		if defaultNameServerIP != nil {
			emptyCoreFile = fmt.Sprintf(expectedEmptyCoreFileWithServer, defaultNameServerIP.String())
		}
		requireFileChanged(ctx, t, corefilePath, emptyCoreFile)

		conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "id",
				Context: &networkservice.ConnectionContext{
					DnsContext: &networkservice.DNSContext{
						Configs: configs,
					},
				},
			},
		})
		require.NoError(t, err)
		requireFileChanged(ctx, t, corefilePath, expectedCoreFile)

		_, err = client.Close(ctx, conn)
		require.NoError(t, err)
		requireFileChanged(ctx, t, corefilePath, emptyCoreFile)
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
	require.FailNowf(t, "fail to wait update", "file has not updated. Last content: %s", r)
}
