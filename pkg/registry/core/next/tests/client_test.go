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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkServiceRegistryClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewRegistryClient().RegisterNSE(context.Context(nil), nil)
	})
}

func TestRegistryClientBranches(t *testing.T) {
	samples := [][]registry.NetworkServiceRegistryClient{
		{visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), visitRegistryClient()},
		{emptyRegistryClient(), visitRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), emptyRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), emptyRegistryClient()},
		{adapters.NewRegistryServerToClient(visitRegistryServer())},
		{adapters.NewRegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer())},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(emptyRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer()), emptyRegistryClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		s := next.NewRegistryClient(sample...)

		ctx := visit(context.Background())
		_, _ = s.RegisterNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.RemoveNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)
	}
}
