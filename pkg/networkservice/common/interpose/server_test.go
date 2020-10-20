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

package interpose_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

func TestInterposeServer(t *testing.T) {
	nseURL := url.URL{Scheme: "tcp", Host: "nse.test"}
	crossNSEURL := url.URL{Scheme: "tcp", Host: "cross-nse.test"}

	var interposeRegistry registry.NetworkServiceEndpointRegistryServer
	interposeServer := interpose.NewServer(&interposeRegistry)

	_, err := interposeRegistry.Register(context.TODO(), &registry.NetworkServiceEndpoint{
		Name: "interpose-nse#",
		Url:  crossNSEURL.String(),
	})
	require.NoError(t, err)

	touchServer := new(touchServer)

	client := next.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		adapters.NewServerToClient(next.NewNetworkServiceServer(
			updatepath.NewServer("nsmgr"),
			clienturl.NewServer(&nseURL),
			interposeServer,
			checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				clientURL := clienturlctx.ClientURL(ctx)
				require.NotNil(t, clientURL)
				require.Equal(t, crossNSEURL, *clientURL)
			}),
		)),
		adapters.NewServerToClient(next.NewNetworkServiceServer(
			updatepath.NewServer("interpose-nse"),
		)),
		adapters.NewServerToClient(next.NewNetworkServiceServer(
			updatepath.NewServer("nsmgr"),
			clienturl.NewServer(&nseURL),
			interposeServer,
			checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				clientURL := clienturlctx.ClientURL(ctx)
				require.NotNil(t, clientURL)
				require.Equal(t, nseURL, *clientURL)
			}),
		)),
		adapters.NewServerToClient(next.NewNetworkServiceServer(
			updatepath.NewServer("endpoint"),
			touchServer,
		)),
	)

	// 1. Request

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	conn, err := client.Request(context.TODO(), request)
	require.NoError(t, err)
	require.True(t, touchServer.touched)

	// 2. Refresh

	request = request.Clone()
	request.Connection = conn.Clone()

	touchServer.touched = false

	conn, err = client.Request(context.TODO(), request)
	require.NoError(t, err)
	require.True(t, touchServer.touched)

	// 3. Close

	conn = conn.Clone()

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

type touchServer struct {
	touched bool
}

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
