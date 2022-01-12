// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package seturl_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func Test_StoreUrlNSEServer(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()

	s := next.NewNetworkServiceEndpointRegistryServer(
		seturl.NewNetworkServiceEndpointRegistryServer(&url.URL{Scheme: "tcp", Host: "127.0.0.1"}),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  "unix://file.sock",
	})
	require.NoError(t, err)

	stream, err := adapters.NetworkServiceEndpointServerToClient(s).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse-1",
		},
	})
	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, list, 1)

	require.Equal(t, "tcp://127.0.0.1", list[0].Url)
}
