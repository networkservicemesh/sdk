// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package heal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestHealClient_FindTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	defer nsmgrCancel()

	fwdName := sandbox.UniqueName("forwarder")
	domain := sandbox.NewBuilder(ctx, t).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, nodeNum int) {
			node.NewNSMgr(nsmgrCtx, sandbox.UniqueName("nsmgr"), nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name:                fwdName,
				NetworkServiceNames: []string{"forwarder"},
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"forwarder": {
						Labels: map[string]string{
							"p2p": "true",
						},
					},
				},
			}, sandbox.GenerateTestToken)
		}).
		Build()

	// 1. Create NS, NSE find clients
	findCtx, findCancel := context.WithCancel(ctx)

	nsRegistryClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithClientURL(sandbox.CloneURL(domain.Nodes[0].NSMgr.URL)),
		registryclient.WithDialOptions(sandbox.DialOptions()...))

	nsRespStream, err := nsRegistryClient.Find(findCtx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
		Watch:          true,
	})
	require.NoError(t, err)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(sandbox.CloneURL(domain.Nodes[0].NSMgr.URL)),
		registryclient.WithDialOptions(sandbox.DialOptions()...))

	nseRespStream, err := nseRegistryClient.Find(findCtx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
		Watch:                  true,
	})
	require.NoError(t, err)

	// 2. Restart NSMgr
	mgr := domain.Nodes[0].NSMgr

	nsmgrCancel()
	require.Eventually(t, func() bool {
		return sandbox.CheckURLFree(mgr.URL)
	}, 2*time.Second, 100*time.Millisecond)

	mgr = domain.Nodes[0].NewNSMgr(ctx, mgr.Name, mgr.URL, sandbox.GenerateTestToken, nsmgr.NewServer)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "ns",
	})
	require.NoError(t, err)

	_, err = nseRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  "tcp://0.0.0.0",
	})
	require.NoError(t, err)

	// 4. Validate NS, NSE streams working
	nsResp, err := nsRespStream.Recv()
	require.NoError(t, err)
	require.Equal(t, "ns", nsResp.GetNetworkService().GetName())

	m := map[string]struct{}{
		"nse":   {},
		fwdName: {},
	}

	nseResp, err := nseRespStream.Recv()
	require.NoError(t, err)
	require.Contains(t, m, nseResp.GetNetworkServiceEndpoint().GetName())
	delete(m, nseResp.GetNetworkServiceEndpoint().GetName())

	nseResp, err = nseRespStream.Recv()
	require.NoError(t, err)
	require.Contains(t, m, nseResp.GetNetworkServiceEndpoint().GetName())

	// 5. Close NS, NSE streams
	findCancel()

	require.Eventually(t, func() bool {
		_, err := nsRespStream.Recv()
		return err != nil
	}, 2*time.Second, time.Millisecond*30)

	require.Eventually(t, func() bool {
		_, err := nseRespStream.Recv()
		return err != nil
	}, 2*time.Second, time.Millisecond*30)
}
