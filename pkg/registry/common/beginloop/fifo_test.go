// Copyright (c) 2023 Cisco and/or its affiliates.
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

package beginloop_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginloop"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/stretchr/testify/require"
)

func TestFIFO(t *testing.T) {
	server := next.NewNetworkServiceEndpointRegistryServer(
		beginloop.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)
	count := 10

	nses := []*registry.NetworkServiceEndpoint{}

	for i := 0; i < count; i++ {
		nses = append(nses, &registry.NetworkServiceEndpoint{Name: "nse", Url: fmt.Sprint(i)})
	}

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		local := i
		go func() {
			_, err := server.Register(begin.WithID(context.Background(), local), nses[local])
			require.NoError(t, err)
			wg.Done()
		}()
		time.Sleep(time.Microsecond * 30)
	}
	wg.Wait()

	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse",
		},
	}
	stream, err := adapters.NetworkServiceEndpointServerToClient(server).Find(context.Background(), query)
	require.NoError(t, err)

	nses = registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, nses, 1)
	require.Equal(t, nses[0].Url, fmt.Sprint(count-1))
}
