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

package memory

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/common/custom"
)

type memoryNetworkServeRegistry struct {
	storage *Storage
	registry.NetworkServiceRegistryServer
}

func (m *memoryNetworkServeRegistry) processRegistration(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	if registration == nil {
		return nil, errors.New("can not register nil registration")
	}
	registration.NetworkServiceEndpoint.State = "RUNNING"

	m.storage.NetworkServiceEndpoints.Store(registration.NetworkServiceEndpoint.Name, registration.NetworkServiceEndpoint)
	m.storage.NetworkServices.Store(registration.NetworkService.Name, registration.NetworkService)
	return registration, nil
}

func (m *memoryNetworkServeRegistry) removeRegistration(ctx context.Context, request *registry.RemoveNSERequest) (*registry.RemoveNSERequest, error) {
	m.storage.NetworkServiceEndpoints.Delete(request.NetworkServiceEndpointName)
	return request, nil
}

// NewNetworkServiceRegistryServer returns new NetworkServiceRegistryServer based on specific resource client
func NewNetworkServiceRegistryServer(storage *Storage) registry.NetworkServiceRegistryServer {
	result := &memoryNetworkServeRegistry{
		storage: storage,
	}
	result.NetworkServiceRegistryServer = custom.NewServer(result.processRegistration, nil, result.removeRegistration)
	return result
}

var _ registry.NetworkServiceRegistryServer = &memoryNetworkServeRegistry{}
