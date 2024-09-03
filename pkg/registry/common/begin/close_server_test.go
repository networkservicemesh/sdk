// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestCloseServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		adapters.NetworkServiceEndpointClientToServer(&markClient{t: t}),
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: id,
	})
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	assert.Equal(t, conn.GetNetworkServiceLabels()[mark].GetLabels()[mark], mark)
	conn = conn.Clone()
	delete(conn.GetNetworkServiceLabels()[mark].GetLabels(), mark)
	assert.Zero(t, conn.GetNetworkServiceLabels()[mark].GetLabels()[mark])
	_, err = server.Unregister(ctx, conn)
	assert.NoError(t, err)
}
