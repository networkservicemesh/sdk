// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package crossnse_test define a full nsmgr + cross connect NSE + Endpoint GRPC based test
package interpose_test

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	adapters2 "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	next_reg "github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interpose"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"

	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectpeer"
	interpose_reg "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

type copyClient struct {
	requests []*networkservice.NetworkServiceRequest
}

func (c *copyClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request = request.Clone()
	c.requests = append(c.requests, request)
	// To be sure it is not modified after and we could check contents.
	request = request.Clone()
	return next.Client(ctx).Request(ctx, request)
}

func (c *copyClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn = conn.Clone()
	return next.Client(ctx).Close(ctx, conn)
}

func TestCrossNSERequest(t *testing.T) {
	var regServer registry.NetworkServiceEndpointRegistryServer
	crossURL := "test://crossconnection"
	clientURL := &url.URL{
		Scheme: "unix",
		Path:   "/var/run/nse-1.sock",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := endpoint.NewServer(ctx, "nsmgr",
		authorize.NewServer(),
		TokenGenerator,
		interpose.NewServer("nsmgr", &regServer))

	regClient := next_reg.NewNetworkServiceEndpointRegistryClient(interpose_reg.NewNetworkServiceEndpointRegistryClient(), adapters2.NetworkServiceEndpointServerToClient(regServer))

	reg, err := regClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "cross-nse",
		Url:  crossURL,
	})
	require.Nil(t, err)
	require.NotNil(t, reg)

	interposeRegName := reg.Name
	interposeNSE := endpoint.NewServer(ctx, interposeRegName,
		authorize.NewServer(),
		TokenGenerator)

	copyClient := &copyClient{}

	client := chain.NewNetworkServiceClient(authorize.NewClient(),
		updatepath.NewClient("client"),
		injectpeer.NewClient(),
		updatetoken.NewClient(TokenGenerator),
		copyClient, // Now we go into server

		adapters.NewServerToClient(
			next.NewNetworkServiceServer(server, checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				require.Equal(t, crossURL, clienturlctx.ClientURL(ctx).String())
			}))),
		copyClient, // Now we go into server
		// We need to call cross nse server here
		// Add one more path with cross NSE
		adapters.NewServerToClient(interposeNSE),

		copyClient, // Now we go into server
		// Go again on server
		adapters.NewServerToClient(next.NewNetworkServiceServer(clienturl.NewServer(clientURL), server, checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.Equal(t, clientURL.String(), clienturlctx.ClientURL(ctx).String())
		}))),
		copyClient, // Now we go into server
	)

	var conn *networkservice.Connection
	conn, err = client.Request(clienturlctx.WithClientURL(context.Background(), clientURL), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "my-service",
		},
	})
	require.Nil(t, err)
	segments := conn.GetPath().GetPathSegments()
	require.NotNil(t, conn)
	require.Equal(t, 4, len(copyClient.requests))
	require.Equal(t, 4, len(segments))
	require.Equal(t, "nsmgr", segments[1].Name)
	require.Equal(t, interposeRegName, segments[2].Name)
	require.Equal(t, "nsmgr", segments[3].Name)
}
