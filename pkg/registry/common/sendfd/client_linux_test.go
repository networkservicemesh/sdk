// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

//+build linux

package sendfd_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func TestSendFDNSEClient_CallsNextUnregister(t *testing.T) {
	unregisterClient := new(unregisterNSEClient)

	c := next.NewNetworkServiceEndpointRegistryClient(
		sendfd.NewNetworkServiceEndpointRegistryClient(),
		unregisterClient,
	)

	_, err := c.Unregister(context.Background(), new(registry.NetworkServiceEndpoint))
	require.NoError(t, err)

	require.True(t, unregisterClient.isUnregistered)
}

type unregisterNSEClient struct {
	isUnregistered bool

	registry.NetworkServiceEndpointRegistryClient
}

func (c *unregisterNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.isUnregistered = true
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
