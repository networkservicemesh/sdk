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
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/swap"
)

func TestNewSwapNetworkServiceRegistryServer(t *testing.T) {
	s := swap.NewNetworkServiceRegistryServer("my.cluster")
	response, err := s.Register(context.Background(), &registry.NetworkService{Name: "my-ns@floating.registry.domain"})
	require.Nil(t, err)
	require.Equal(t, "my-ns@my.cluster", response.Name)
	request := &registry.NetworkService{Name: "my-ns@floating.registry.domain"}
	_, err = s.Unregister(context.Background(), request)
	require.Nil(t, err)
	require.Equal(t, "my-ns@my.cluster", request.Name)
}
