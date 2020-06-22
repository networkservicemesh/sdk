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

package aggregate_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/core/aggregate"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

func TestNewNetworkServiceEndpointFindClient(t *testing.T) {
	defer goleak.VerifyNone(t)
	var clients []registry.NetworkServiceEndpointRegistry_FindClient
	for i := 0; i < 10; i++ {
		ch := make(chan *registry.NetworkServiceEndpoint)
		clients = append(clients, streamchannel.NewNetworkServiceEndpointFindClient(context.Background(), ch))
		go func() {
			for j := 0; j < 10; j++ {
				ch <- &registry.NetworkServiceEndpoint{}
			}
			close(ch)
		}()
	}
	c := aggregate.NewNetworkServiceEndpointFindClient(clients...)
	list := registry.ReadNetworkServiceEndpointList(c)
	require.Len(t, list, 100)
}
