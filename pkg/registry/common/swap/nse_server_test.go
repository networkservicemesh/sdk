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

package swap_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/swap"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

func TestNewSwapNetworkServiceEndpointRegistryServer_RegisterUnregister(t *testing.T) {
	s := swap.NewNetworkServiceEndpointRegistryServer("my.cluster", nil)
	response, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "my-nse@floating.registry.domain"})
	require.Nil(t, err)
	require.Equal(t, response.Name, "my-nse@my.cluster")
	request := &registry.NetworkServiceEndpoint{Name: "my-nse@floating.registry.domain"}
	_, err = s.Unregister(context.Background(), request)
	require.Nil(t, err)
	require.Equal(t, "my-nse@my.cluster", request.Name)
}

func TestNewSwapNetworkServiceEndpointRegistryServer_Find(t *testing.T) {
	proxyNSMgr := &url.URL{Path: "proxy"}
	mem := memory.NewNetworkServiceEndpointRegistryServer()
	s := next.NewNetworkServiceEndpointRegistryServer(
		swap.NewNetworkServiceEndpointRegistryServer("my.cluster", proxyNSMgr),
		mem,
	)
	_, err := mem.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1@remote.domain",
		Url:  "remote_nsmgr_url",
	})
	require.Nil(t, err)
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
	err = s.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse-1"}}, streamchannel.NewNetworkServiceEndpointFindServer(context.Background(), ch))
	require.Nil(t, err)
	findResult := <-ch
	require.Equal(t, interdomain.Join("nse-1", "remote_nsmgr_url"), findResult.Name)
	require.Equal(t, proxyNSMgr.String(), findResult.Url)
}
