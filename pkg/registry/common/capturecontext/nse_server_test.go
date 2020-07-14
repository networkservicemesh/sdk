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

package capturecontext_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"

	"github.com/networkservicemesh/sdk/pkg/registry/common/capturecontext"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

func TestNSEServerContextStorage(t *testing.T) {
	chain := next.NewNetworkServiceEndpointRegistryServer(adapters.NetworkServiceEndpointClientToServer(&writeNSEClient{}), capturecontext.NewNetworkServiceEndpointRegistryServer(), adapters.NetworkServiceEndpointClientToServer(&checkContextNSEClient{}))

	ctx := context.WithValue(context.Background(), writerNSEKey, true)
	_, err := chain.Register(ctx, nil)
	require.NoError(t, err)

	err = chain.Find(nil, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, nil))
	require.NoError(t, err)

	_, err = chain.Unregister(ctx, nil)
	require.NoError(t, err)
}
