// Copyright (c) 2020-2023 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

const (
	// Tick is a default tick for the sandbox tests
	Tick = 10 * time.Millisecond
	// Timeout is a default timeout for the sandbox tests
	Timeout = 10 * time.Second
)

// CheckSecondRequestsReceived makes checking multiple requests for the sandbox tests
func CheckSecondRequestsReceived(requestsDone func() int) func() bool {
	return func() bool {
		return requestsDone() >= 2
	}
}

// RequireIPv4Lookup makes IPv4 lookup for the sandbox tests
func RequireIPv4Lookup(ctx context.Context, t *testing.T, r *net.Resolver, host, expected string) {
	addrs, err := r.LookupIP(ctx, "ip4", host)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	require.Equal(t, expected, addrs[0].String())
}

// DefaultRegistryService is a default registry service for the sandbox tests
func DefaultRegistryService(name string) *registry.NetworkService {
	return &registry.NetworkService{
		Name: name,
	}
}

// DefaultRegistryEndpoint returns a default registry endpoint for the sandbox tests
func DefaultRegistryEndpoint(nsName string) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsName},
	}
}

// DefaultRequest returns a default request to Network Service for the sandbox tests
func DefaultRequest(nsName string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: nsName,
			Context:        &networkservice.ConnectionContext{},
			Labels:         make(map[string]string),
		},
	}
}
