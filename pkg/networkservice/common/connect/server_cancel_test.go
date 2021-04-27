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

package connect_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestConnect_CancelDuringRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	domain := sandbox.NewBuilder(t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()

	service1Name := "my-service-endpoint"
	service2Name := "my-service-with-passthrough"

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "endpoint-1",
		NetworkServiceNames: []string{service1Name},
	}
	nscCtx, nscCancel := context.WithCancel(ctx)
	var flag atomic.Bool
	_, err = domain.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken, checkrequest.NewServer(t, func(*testing.T, *networkservice.NetworkServiceRequest) {
		if flag.Load() {
			nscCancel()
		}
	}))
	require.NoError(t, err)

	var counter atomic.Int32
	ptClient := newPassTroughClient(service1Name)
	kernelClient := kernel.NewClient()
	clientName := fmt.Sprintf("connectClient-%v", uuid.New().String())
	clientFactory := func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		counter.Add(1)
		return chain.NewNetworkServiceClient(
			mechanismtranslation.NewClient(),
			client.NewClient(ctx, cc, client.WithName(clientName),
				client.WithAdditionalFunctionality(ptClient, kernelClient)),
		)
	}

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "endpoint-2",
		NetworkServiceNames: []string{service2Name},
	}
	_, err = domain.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken,
		chain.NewNetworkServiceServer(
			clienturl.NewServer(domain.Nodes[0].NSMgr.URL),
			connect.NewServer(ctx,
				clientFactory,
				connect.WithDialOptions(sandbox.DefaultDialOptions(sandbox.GenerateTestToken)...),
			),
		),
	)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: service2Name,
			Context:        &networkservice.ConnectionContext{},
		},
	}
	nsc1 := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	nsc2 := domain.Nodes[0].NewClient(nscCtx, sandbox.GenerateTestToken)

	_, err = nsc1.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, int32(1), counter.Load())

	flag.Store(true)
	_, err = nsc2.Request(ctx, request.Clone())
	require.Error(t, err)
	require.Equal(t, int32(1), counter.Load())

	_, err = nsc1.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, int32(1), counter.Load())
}

type passThroughClient struct {
	networkService string
}

func newPassTroughClient(networkService string) *passThroughClient {
	return &passThroughClient{
		networkService: networkService,
	}
}

func (c *passThroughClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.NetworkService = c.networkService
	request.Connection.NetworkServiceEndpointName = ""
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *passThroughClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.NetworkService = c.networkService
	return next.Client(ctx).Close(ctx, conn, opts...)
}
