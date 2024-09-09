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

package setpayload_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setpayload"

	"github.com/stretchr/testify/require"
)

func TestSetPayload_Empty(t *testing.T) {
	server := setpayload.NewNetworkServiceRegistryServer()
	ns := &registry.NetworkService{}

	reg, err := server.Register(context.Background(), ns)
	require.NoError(t, err)

	require.Equal(t, payload.IP, ns.GetPayload())

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func TestSetPayload_Ethernet(t *testing.T) {
	server := setpayload.NewNetworkServiceRegistryServer()
	ns := &registry.NetworkService{Payload: payload.Ethernet}

	reg, err := server.Register(context.Background(), ns)
	require.NoError(t, err)

	require.Equal(t, payload.Ethernet, ns.GetPayload())

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func TestSetPayload_WithCustomPayload(t *testing.T) {
	testPayload := "some payload"

	server := setpayload.NewNetworkServiceRegistryServer(setpayload.WithPayload(testPayload))
	ns := &registry.NetworkService{Payload: ""}

	reg, err := server.Register(context.Background(), ns)
	require.NoError(t, err)

	require.Equal(t, testPayload, reg.GetPayload())

	_, err = server.Unregister(context.Background(), reg)
	require.NoError(t, err)
}
