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

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"testing"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

func TestNewNetworkServiceServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewRegistryServer().RegisterNSE(context.Context(nil), nil)
	})
}

func TestServerBranches(t *testing.T) {
	servers := [][]registry.NetworkServiceRegistryServer{
		{visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), visitRegistryServer()},
		{emptyRegistryServer(), visitRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), emptyRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), emptyRegistryServer()},
		{adapters.NewRegistryClientToServer(visitRegistryClient())},
		{adapters.NewRegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient())},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(emptyRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient()), emptyRegistryServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2}
	for i, sample := range servers {
		s := next.NewRegistryServer(sample...)

		ctx := visit(context.Background())
		_, _ = s.RegisterNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_, _ = s.RemoveNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}
