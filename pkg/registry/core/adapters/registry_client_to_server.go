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

package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"
)

// NewRegistryClientToServer - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client
func NewRegistryClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return NewRegistryClientToServerNext(client, nil)
}

type registryClientToServer struct {
	client registry.NetworkServiceRegistryClient
	next   func(ctx context.Context) registry.NetworkServiceRegistryServer
}

// NewRegistryClientToServerNext - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client
func NewRegistryClientToServerNext(client registry.NetworkServiceRegistryClient, next func(ctx context.Context) registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &registryClientToServer{client: client, next: next}
}

func (r *registryClientToServer) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	result, err := r.client.RegisterNSE(ctx, registration)
	if err != nil || r.next == nil {
		return nil, err
	}
	return r.next(ctx).RegisterNSE(ctx, result)
}

// BulkRegisterNSE - register NSEs in a Bulk
func (r *registryClientToServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	client, err := r.client.BulkRegisterNSE(server.Context())
	if err != nil {
		return err
	}

	// Handle server
	if client != nil {
		go func() {
			for {
				reg, err := client.Recv()
				if err != nil {
					logrus.Errorf("Error in BulkRegisterNSE %v", err)
					return
				}
				if reg == nil {
					break
				}
				err = server.Send(reg)
				if err != nil {
					logrus.Errorf("Error in BulkRegisterNSE %v", err)
					return
				}
			}
		}()
	}
	return nil
}

func (r *registryClientToServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	empl, err := r.client.RemoveNSE(ctx, request)
	if err != nil || r.next == nil {
		return empl, err
	}
	return r.next(ctx).RemoveNSE(ctx, request)
}

// Implementation check
var _ registry.NetworkServiceRegistryServer = &registryClientToServer{}
