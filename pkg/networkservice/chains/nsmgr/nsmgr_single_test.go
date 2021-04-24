// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package nsmgr_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_DNSUsecase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var cluster = sandbox.NewBuilder(t).SetNodesCount(1).SetNSMgrProxySupplier(nil).SetRegistryProxySupplier(nil).SetContext(ctx).Build()

	_, err := cluster.Nodes[0].NewEndpoint(ctx, defaultRegistryEndpoint(), sandbox.GenerateTestToken, dnscontext.NewServer(
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.8.8"},
			SearchDomains: []string{"my.domain1"},
		},
		&networkservice.DNSConfig{
			DnsServerIps:  []string{"8.8.4.4"},
			SearchDomains: []string{"my.domain1"},
		},
	))
	require.NoError(t, err)

	corefilePath := filepath.Join(t.TempDir(), "corefile")
	resolveConfigPath := filepath.Join(t.TempDir(), "resolv.conf")

	err = ioutil.WriteFile(resolveConfigPath, []byte("nameserver 8.8.4.4\nsearch example.com\n"), os.ModePerm)
	require.NoError(t, err)

	const expectedCorefile = ". {\n\tforward . 8.8.4.4\n\tlog\n\treload\n}\nmy.domain1 {\n\tfanout . 8.8.4.4 8.8.8.8\n\tlog\n}"

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(cluster.Nodes[0].NSMgr.URL), sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = cc.Close()
	})

	c := client.NewClient(
		ctx,
		cc,
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(
			dnscontext.NewClient(
				dnscontext.WithChainContext(ctx),
				dnscontext.WithCorefilePath(corefilePath),
				dnscontext.WithResolveConfigPath(resolveConfigPath),
			),
		),
	)

	resp, err := c.Request(ctx, defaultRequest())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		// #nosec
		b, readFileErr := ioutil.ReadFile(corefilePath)
		if readFileErr != nil {
			return false
		}
		return string(b) == expectedCorefile
	}, time.Second, time.Millisecond*100)
	_, err = c.Close(ctx, resp)
	require.NoError(t, err)
	_, err = cluster.Nodes[0].EndpointRegistryClient.Unregister(ctx, defaultRegistryEndpoint())
	require.NoError(t, err)
}
