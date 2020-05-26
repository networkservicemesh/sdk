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

// Package nsmgr - provide Network Service Mesh manager registry tracker, to update endpoint registrations.
package nsmgr

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type nsmRegistryServer struct {
	registry.NetworkServiceRegistryServer
	serialize.Executor
	manager *registry.NetworkServiceManager
}

// NewServer - construct a new NSM aware NSE registry server
// Server is using latest NSM registration and update endpoints accordingly.
func NewServer(manager *registry.NetworkServiceManager) registry.NetworkServiceRegistryServer {
	return &nsmRegistryServer{
		manager: manager,
	}
}

// RegisterNSE - register NSE with current manager associated.
func (n *nsmRegistryServer) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	// update manager to be proper one.
	<-n.AsyncExec(func() {
		registration.NetworkServiceManager = n.manager
	})
	result, err := next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, registration)
	if result != nil {
		<-n.AsyncExec(func() {
			result.NetworkServiceManager = n.manager
		})
	}
	return result, err
}

type nsmBulkRegServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	server registry.NetworkServiceRegistry_BulkRegisterNSEServer
	ctx    context.Context
	n      *nsmRegistryServer
}

func (ts *nsmBulkRegServer) Send(reg *registry.NSERegistration) error {
	reg.NetworkServiceManager = ts.n.manager
	err := ts.server.Send(reg)
	return err
}
func (ts *nsmBulkRegServer) Recv() (*registry.NSERegistration, error) {
	reg, err := ts.server.Recv()
	if reg != nil {
		reg.NetworkServiceManager = ts.n.manager
	}
	return reg, err
}

func (ts *nsmBulkRegServer) Context() context.Context {
	return ts.ctx
}

// BulkRegisterNSE - register a stream of NSEs
func (n *nsmRegistryServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	ctx := server.Context()
	brs := &nsmBulkRegServer{
		ctx:    server.Context(),
		server: server,
		n:      n,
	}
	return next.NetworkServiceRegistryServer(ctx).BulkRegisterNSE(brs)
}

// RemoveNSE - remove NSE previously registered.
func (n *nsmRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, request)
}

// Compile level checks.
var _ registry.NetworkServiceRegistryServer = &nsmRegistryServer{}
