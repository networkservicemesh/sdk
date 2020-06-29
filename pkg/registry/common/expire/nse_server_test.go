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

package expire_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func TestNewNetworkServiceEndpointRegistryServer(t *testing.T) {
	s := expire.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer(), expire.WithPeriod(testPeriod))
	expiration := time.Now().Add(testPeriod * 2)
	_, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		ExpirationTime: &timestamp.Timestamp{
			Seconds: expiration.Unix(),
			Nanos:   int32(expiration.Nanosecond()),
		},
	})
	require.Nil(t, err)
	c := adapters.NetworkServiceEndpointServerToClient(s)
	stream, err := c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
	})
	require.Nil(t, err)
	list := registry.ReadNetworkServiceEndpointList(stream)
	require.NotEmpty(t, list)
	require.Eventually(t, func() bool {
		stream, err = c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
		})
		require.Nil(t, err)
		list = registry.ReadNetworkServiceEndpointList(stream)
		return len(list) == 0
	}, time.Second, time.Millisecond*100)
}
