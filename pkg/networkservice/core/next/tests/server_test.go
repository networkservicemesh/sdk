// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

func TestNewNetworkServiceServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceServer().Request(context.Context(nil), nil)
		_, _ = next.NewWrappedNetworkServiceServer(func(server networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
			return server
		}).Request(context.Context(nil), nil)
	})
}

func TestServerBranches(t *testing.T) {
	servers := [][]networkservice.NetworkServiceServer{
		{visitServer()},
		{visitServer(), visitServer()},
		{visitServer(), visitServer(), visitServer()},
		{emptyServer(), visitServer(), visitServer()},
		{visitServer(), emptyServer(), visitServer()},
		{visitServer(), visitServer(), emptyServer()},
		{adapters.NewClientToServer(visitClient())},
		{adapters.NewClientToServer(visitClient()), visitServer()},
		{visitServer(), adapters.NewClientToServer(visitClient()), visitServer()},
		{visitServer(), visitServer(), adapters.NewClientToServer(visitClient())},
		{visitServer(), adapters.NewClientToServer(emptyClient()), visitServer()},
		{visitServer(), adapters.NewClientToServer(visitClient()), emptyServer()},
		{visitServer(), next.NewNetworkServiceServer(next.NewNetworkServiceServer(visitServer())), visitServer()},
		{visitServer(), next.NewNetworkServiceServer(next.NewNetworkServiceServer(emptyServer())), visitServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2, 3, 1}
	for i, sample := range servers {
		ctx := visit(context.Background())
		s := next.NewNetworkServiceServer(sample...)
		_, _ = s.Request(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
		ctx = visit(context.Background())
		_, _ = s.Close(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}
