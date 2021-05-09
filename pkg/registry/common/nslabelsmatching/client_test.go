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

package nslabelsmatching_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/nslabelsmatching"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	testEnvName                      = "NODE_NAME"
	testEnvValue                     = "testValue"
	destinationTestKey               = "nodeName"
	destinationTestCorrectTemplate   = "{{index .Src \"NodeNameKey\"}}"
	destinationTestIncorrectTemplate = "{{index .Src \"SomeRandomKey\"}}"
	destinationTestNotATemplate      = "SomeRandomString"
)

func TestCorrectTemplate(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	domain := sandbox.NewBuilder(t).
		SetNodesCount(0).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetContext(ctx).
		Build()

	// start grpc client connection and register it
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain.Registry.URL), sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...)
	require.NoError(t, err)
	defer func() {
		_ = cc.Close()
	}()

	nsrc := chain.NewNetworkServiceRegistryClient(
		nslabelsmatching.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(cc),
	)

	ns := &registry.NetworkService{
		Matches: []*registry.Match{
			{
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							destinationTestKey: destinationTestCorrectTemplate,
						},
					},
				},
			},
		},
	}

	ns, err = nsrc.Register(ctx, ns)
	require.NoError(t, err)

	nsrfc, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: ns})
	require.NoError(t, err)

	ns, err = nsrfc.Recv()
	require.NoError(t, err)
	require.Equal(t, testEnvValue, ns.Matches[0].Routes[0].DestinationSelector[destinationTestKey])

	require.NoError(t, ctx.Err())
}

func TestIncorrectTemplate(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	domain := sandbox.NewBuilder(t).
		SetNodesCount(0).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetContext(ctx).
		Build()

	// start grpc client connection and register it
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain.Registry.URL), sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...)
	require.NoError(t, err)
	defer func() {
		_ = cc.Close()
	}()

	nsrc := chain.NewNetworkServiceRegistryClient(
		nslabelsmatching.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(cc),
	)

	ns := &registry.NetworkService{
		Matches: []*registry.Match{
			{
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							destinationTestKey: destinationTestIncorrectTemplate,
						},
					},
				},
			},
		},
	}

	ns, err = nsrc.Register(ctx, ns)
	require.NoError(t, err)

	nsrfc, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: ns})
	require.NoError(t, err)

	ns, err = nsrfc.Recv()
	require.NoError(t, err)
	require.NotEqual(t, testEnvValue, ns.Matches[0].Routes[0].DestinationSelector[destinationTestKey])
	require.True(t, ns.Matches[0].Routes[0].DestinationSelector[destinationTestKey] == "")

	require.NoError(t, ctx.Err())
}

func TestNotATemplate(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	domain := sandbox.NewBuilder(t).
		SetNodesCount(0).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetContext(ctx).
		Build()

	// start grpc client connection and register it
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain.Registry.URL), sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...)
	require.NoError(t, err)
	defer func() {
		_ = cc.Close()
	}()

	nsrc := chain.NewNetworkServiceRegistryClient(
		nslabelsmatching.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(cc),
	)

	ns := &registry.NetworkService{
		Matches: []*registry.Match{
			{
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							destinationTestKey: destinationTestNotATemplate,
						},
					},
				},
			},
		},
	}

	ns, err = nsrc.Register(ctx, ns)
	require.NoError(t, err)

	nsrfc, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: ns})
	require.NoError(t, err)

	ns, err = nsrfc.Recv()
	require.NoError(t, err)
	require.Equal(t, destinationTestNotATemplate, ns.Matches[0].Routes[0].DestinationSelector[destinationTestKey])

	require.NoError(t, ctx.Err())
}
