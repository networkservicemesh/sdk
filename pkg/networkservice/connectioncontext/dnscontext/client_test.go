// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

package dnscontext_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_DNSContextClient_Usecases(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err := os.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\nsearch example.com\n"), os.ModePerm)
	require.NoError(t, err)

	dnsConfigMap := new(genericsync.Map[string, []*networkservice.DNSConfig])
	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		dnscontext.NewClient(
			dnscontext.WithChainContext(ctx),
			dnscontext.WithResolveConfigPath(resolveConfigPath),
			dnscontext.WithDNSConfigsMap(dnsConfigMap),
		),
	)

	const expectedResolvconfFile = `#nameserver 8.8.4.4
#search example.com
#

nameserver 127.0.0.1`

	requireFileChanged(ctx, t, resolveConfigPath, expectedResolvconfFile)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:      "nsc-1",
			Context: &networkservice.ConnectionContext{},
		},
	}

	resp, err := client.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, resp.GetContext().GetDnsContext())
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), 0)
	// Check updated dnsConfigMap
	loadedDNSConfig, ok := dnsConfigMap.Load(resp.GetId())
	require.True(t, ok)
	require.Contains(t, loadedDNSConfig[0].GetDnsServerIps(), "8.8.4.4")
	require.Contains(t, loadedDNSConfig[0].GetSearchDomains(), "example.com")
	_, err = client.Close(ctx, resp)
	require.NoError(t, err)
}

func requireFileChanged(ctx context.Context, t *testing.T, location, expected string) {
	var r string
	for ctx.Err() == nil {
		b, err := os.ReadFile(filepath.Clean(location))
		r = string(b)
		if err == nil && r == expected {
			return
		}
		runtime.Gosched()
	}
	require.FailNowf(t, "fail to wait update", "file has not updated. Last content: %s, expected: %s", r, expected)
}
