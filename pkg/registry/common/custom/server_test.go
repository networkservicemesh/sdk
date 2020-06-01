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

package custom_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/registry/common/custom"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

func TestRegisterCustomNSE(t *testing.T) {
	mgr := &registry.NetworkServiceManager{
		Name: "nsm-1",
		Url:  "tcp://127.0.0.1:5000",
	}
	nseReg := chain.NewNetworkServiceRegistryServer(custom.NewServer(beforeReg, afterReg, nil))
	regEndpoint, err := nseReg.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "my-service",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:               "my-endpoint",
			NetworkServiceName: "my-service",
			Url:                "tcp://127.0.0.1:5001",
		},
	})
	require.Nil(t, err)
	require.NotNil(t, regEndpoint)
	require.Equal(t, mgr.Name, regEndpoint.NetworkServiceManager.Name)
	require.Equal(t, "service-1", regEndpoint.NetworkService.Name)
}

func afterReg(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	registration.NetworkService = &registry.NetworkService{Name: "service-1"}
	return registration, nil
}

func beforeReg(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	registration.NetworkServiceManager = &registry.NetworkServiceManager{Name: "nsm-1"}
	return registration, nil
}

func TestRegisterCustomNSEStream(t *testing.T) {
	mgr := &registry.NetworkServiceManager{
		Name: "nsm-1",
		Url:  "tcp://127.0.0.1:5000",
	}
	nseReg := chain.NewNetworkServiceRegistryServer(custom.NewServer(beforeReg, afterReg, nil))

	client := adapters.NewRegistryServerToClient(nseReg)

	regClient, err := client.BulkRegisterNSE(context.Background())
	require.Nil(t, err)

	err = regClient.Send(&registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "my-service",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:               "my-endpoint",
			NetworkServiceName: "my-service",
			Url:                "tcp://127.0.0.1:5001",
		},
	})
	require.Nil(t, err)
	var regEndpoint *registry.NSERegistration
	regEndpoint, err = regClient.Recv()
	require.Nil(t, err)
	require.NotNil(t, regEndpoint)
	require.Equal(t, mgr.Name, regEndpoint.NetworkServiceManager.Name)
	require.Equal(t, "service-1", regEndpoint.NetworkService.Name)
}
