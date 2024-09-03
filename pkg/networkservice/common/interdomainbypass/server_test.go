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

package interdomainbypass_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interdomainbypass"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

func Test_StoreUrlNSEServer(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()

	var storeRegServer registry.NetworkServiceEndpointRegistryServer

	s := chain.NewNetworkServiceServer(
		interdomainbypass.NewServer(&storeRegServer, new(url.URL)),
		checkcontext.NewServer(t, func(t *testing.T, c context.Context) {
			v := clienturlctx.ClientURL(c)
			require.NotNil(t, v)
			require.Equal(t, url.URL{Scheme: "unix", Host: "file.sock"}, *v)
		}),
	)

	registryServer := next.NewNetworkServiceEndpointRegistryServer(
		storeRegServer,
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	_, err := registryServer.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  "unix://file.sock",
	})
	require.NoError(t, err)

	stream, err := adapters.NetworkServiceEndpointServerToClient(registryServer).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse-1",
		},
	})
	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, list, 1)

	_, err = s.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: "nse-1",
		},
	})

	require.NoError(t, err)
}
