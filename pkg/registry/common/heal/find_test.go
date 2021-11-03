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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	defer nsmgrCancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, nodeNum int) {
			node.NewNSMgr(nsmgrCtx, sandbox.UniqueName("nsmgr"), nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.NetworkServiceEndpoint{
				Name: sandbox.UniqueName("forwarder"),
			}, sandbox.GenerateTestToken)
		}).
		Build()

	// 1. Create NS, NSE find clients
	findCtx, findCancel := context.WithCancel(ctx)

	nsRegistryClient := registryclient.NewNetworkServiceRegistryClient(ctx, sandbox.CloneURL(domain.Nodes[0].NSMgr.URL),
		registryclient.WithDialOptions(sandbox.DialOptions()...))

	nsRespStream, err := nsRegistryClient.Find(findCtx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
		Watch:          true,
	})
	require.NoError(t, err)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx, sandbox.CloneURL(domain.Nodes[0].NSMgr.URL),
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
	}, time.Second, 100*time.Millisecond)

	mgr = domain.Nodes[0].NewNSMgr(ctx, mgr.Name, mgr.URL, sandbox.GenerateTestToken, nsmgr.NewServer)

	// 3. Register new NS, NSE
	_, err = mgr.NetworkServiceRegistryServer().Register(ctx, &registry.NetworkService{
		Name: "ns",
	})
	require.NoError(t, err)

	_, err = mgr.NetworkServiceEndpointRegistryServer().Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  "tcp://0.0.0.0",
	})
	require.NoError(t, err)

	// 4. Validate NS, NSE streams working
	nsResp, err := nsRespStream.Recv()
	require.NoError(t, err)
	require.Equal(t, "ns", nsResp.NetworkService.Name)

	nseResp, err := nseRespStream.Recv()
	require.NoError(t, err)
	require.Equal(t, "nse", nseResp.NetworkServiceEndpoint.Name)

	// 5. Close NS, NSE streams
	findCancel()

	// 6. Validate NS, NSE streams closed
	timer := time.AfterFunc(100*time.Millisecond, t.FailNow)

	_, err = nsRespStream.Recv()
	require.Error(t, err)

	_, err = nseRespStream.Recv()
	require.Error(t, err)

	timer.Stop()
}
