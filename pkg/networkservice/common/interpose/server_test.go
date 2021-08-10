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

package interpose_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	registryinterpose "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	registryadapters "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	registrynext "github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

func TestInterposeServer(t *testing.T) {
	nseURL := url.URL{Scheme: "tcp", Host: "nse.test"}
	crossNSEURL := url.URL{Scheme: "tcp", Host: "cross-nse.test"}

	var interposeRegistry registry.NetworkServiceEndpointRegistryServer
	interposeServer := interpose.NewServer(&interposeRegistry)

	_, err := registrynext.NewNetworkServiceEndpointRegistryClient(
		registryinterpose.NewNetworkServiceEndpointRegistryClient(),
		registryadapters.NetworkServiceEndpointServerToClient(interposeRegistry),
	).Register(context.TODO(), &registry.NetworkServiceEndpoint{
		Name: "forwarder",
		Url:  crossNSEURL.String(),
	})
	require.NoError(t, err)

	counter := new(count.Server)

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
			updatepath.NewServer("forwarder"),
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
			counter,
		)),
	)

	// 1. Request

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	conn, err := client.Request(context.TODO(), request)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Requests())

	// 2. Refresh

	request = request.Clone()
	request.Connection = conn.Clone()

	conn, err = client.Request(context.TODO(), request)
	require.NoError(t, err)
	require.Equal(t, 2, counter.Requests())

	// 3. Close

	conn = conn.Clone()

	_, err = client.Close(context.TODO(), conn)
	require.NoError(t, err)
	require.Equal(t, 1, counter.Closes())
}
