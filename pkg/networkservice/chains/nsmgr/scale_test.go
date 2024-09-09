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

package nsmgr_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestCreateEndpointDuringRequest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "ns-1",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{DestinationSelector: map[string]string{"nodeName": "node[1]"}},
					{DestinationSelector: map[string]string{"create-nse": "true"}},
				},
			},
		},
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "nse-maker",
		NetworkServiceNames: []string{nsReg.GetName()},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			nsReg.GetName(): {Labels: map[string]string{"create-nse": "true"}},
		},
	}

	flag := atomic.Bool{}
	requestCounter := new(count.Server)

	makerServer := &nseMaker{
		ctx:    ctx,
		domain: domain,
		nseReg: &registry.NetworkServiceEndpoint{
			Name:                "nse-impl",
			NetworkServiceNames: []string{nsReg.GetName()},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				nsReg.GetName(): {Labels: map[string]string{"nodeName": "node[1]"}},
			},
		},
		serverFactory: func() networkservice.NetworkServiceServer {
			// we can't use require false here
			// because there will be several calls, and it's expected behavior
			if flag.Swap(true) {
				return nil
			}

			return requestCounter
		},
	}

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, makerServer)

	nsc := domain.Nodes[1].NewClient(ctx, sandbox.GenerateTestToken)

	// check first request
	conn, err := nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsReg.GetName(),
		},
	})
	require.NoError(t, err)
	require.True(t, flag.Load())
	require.Equal(t, 1, requestCounter.Requests())

	// check refresh
	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn,
	})
	require.NoError(t, err)
	require.Equal(t, 2, requestCounter.Requests())

	// check new request
	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsReg.GetName(),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 3, requestCounter.Requests())
}

type nseMaker struct {
	ctx           context.Context
	domain        *sandbox.Domain
	nseReg        *registry.NetworkServiceEndpoint
	serverFactory func() networkservice.NetworkServiceServer
}

func (m *nseMaker) Request(_ context.Context, _ *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	endpoint := m.serverFactory()
	if endpoint == nil {
		return nil, errors.New("can't create new endpoint")
	}

	m.domain.Nodes[1].NewEndpoint(m.ctx, m.nseReg, sandbox.GenerateTestToken, endpoint)

	return nil, errors.New("can't provide requested network service")
}

func (m *nseMaker) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}
