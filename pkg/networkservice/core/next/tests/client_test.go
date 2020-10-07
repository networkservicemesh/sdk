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

package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestNewNetworkServiceClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceClient().Request(context.Context(nil), nil)
		_, _ = next.NewWrappedNetworkServiceClient(func(client networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
			return client
		}).Request(context.Context(nil), nil)
	})
}

func TestClientBranches(t *testing.T) {
	samples := [][]networkservice.NetworkServiceClient{
		{visitClient()},
		{visitClient(), visitClient()},
		{visitClient(), visitClient(), visitClient()},
		{emptyClient(), visitClient(), visitClient()},
		{visitClient(), emptyClient(), visitClient()},
		{visitClient(), visitClient(), emptyClient()},
		{adapters.NewServerToClient(visitServer())},
		{adapters.NewServerToClient(visitServer()), visitClient()},
		{visitClient(), adapters.NewServerToClient(visitServer()), visitClient()},
		{visitClient(), visitClient(), adapters.NewServerToClient(visitServer())},
		{visitClient(), adapters.NewServerToClient(emptyServer()), visitClient()},
		{visitClient(), adapters.NewServerToClient(visitServer()), emptyClient()},
		{visitClient(), next.NewNetworkServiceClient(next.NewNetworkServiceClient(visitClient())), visitClient()},
		{visitClient(), next.NewNetworkServiceClient(next.NewNetworkServiceClient(emptyClient())), visitClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2, 3, 1}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		ctx := visit(context.Background())
		s := next.NewNetworkServiceClient(sample...)
		_, _ = s.Request(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)
		ctx = visit(context.Background())
		_, _ = s.Close(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)
	}
}
