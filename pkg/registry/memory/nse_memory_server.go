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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type memoryNetworkServeRegistry struct {
	storage *Storage
}

func (m *memoryNetworkServeRegistry) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	nseRegistration, err := m.processRegistration(registration)
	if err != nil {
		return nseRegistration, err
	}

	return next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, registration)
}

func (m *memoryNetworkServeRegistry) processRegistration(registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	if registration == nil {
		return nil, errors.New("can not register nil registration")
	}
	registration.NetworkServiceEndpoint.State = "RUNNING"

	m.storage.NetworkServiceEndpoints.Store(registration.NetworkServiceEndpoint.Name, registration.NetworkServiceEndpoint)
	m.storage.NetworkServices.Store(registration.NetworkService.Name, registration.NetworkService)
	return registration, nil
}

type nsmBulkRegServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	server registry.NetworkServiceRegistry_BulkRegisterNSEServer
	m      *memoryNetworkServeRegistry
}

func (ts *nsmBulkRegServer) Send(reg *registry.NSERegistration) error {
	nseRegistration, err := ts.m.processRegistration(reg)
	if err != nil {
		return err
	}
	return ts.server.Send(nseRegistration)
}
func (ts *nsmBulkRegServer) Recv() (*registry.NSERegistration, error) {
	return ts.server.Recv()
}

func (ts *nsmBulkRegServer) Context() context.Context {
	return ts.server.Context()
}

func (m *memoryNetworkServeRegistry) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	bulkRegS := &nsmBulkRegServer{
		m:      m,
		server: s,
	}
	return next.NetworkServiceRegistryServer(s.Context()).BulkRegisterNSE(bulkRegS)
}

func (m *memoryNetworkServeRegistry) RemoveNSE(ctx context.Context, req *registry.RemoveNSERequest) (*empty.Empty, error) {
	m.storage.NetworkServiceEndpoints.Delete(req.NetworkServiceEndpointName)
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, req)
}

// NewNetworkServiceRegistryServer returns new NetworkServiceRegistryServer based on specific resource client
func NewNetworkServiceRegistryServer(storage *Storage) registry.NetworkServiceRegistryServer {
	return &memoryNetworkServeRegistry{
		storage: storage,
	}
}

var _ registry.NetworkServiceRegistryServer = &memoryNetworkServeRegistry{}
