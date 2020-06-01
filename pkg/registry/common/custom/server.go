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

package custom

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

// RegFunction - operation to perform on NSE registration.
type RegFunction func(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error)

// RemoveFunction - operation to perform on NSE registration.
type RemoveFunction func(ctx context.Context, request *registry.RemoveNSERequest) (*registry.RemoveNSERequest, error)

type customRegServer struct {
	serialize.Executor
	before RegFunction
	after  RegFunction
	remove RemoveFunction
}

// NewServer - create NSE update reg server
func NewServer(before, after RegFunction, remove RemoveFunction) registry.NetworkServiceRegistryServer {
	return &customRegServer{
		before: before,
		after:  after,
		remove: remove,
	}
}

// RegisterNSE - register NSE with current manager associated.
func (n *customRegServer) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (result *registry.NSERegistration, err error) {
	result = registration
	if n.before != nil {
		result, err = n.before(ctx, result)
		if err != nil {
			return nil, err
		}
	}
	result, err = next.NetworkServiceRegistryServer(ctx).RegisterNSE(ctx, result)
	if err != nil {
		return nil, err
	}
	if n.after != nil {
		result, err = n.after(ctx, result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

type nsmBulkRegServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	server registry.NetworkServiceRegistry_BulkRegisterNSEServer
	ctx    context.Context
	n      *customRegServer
}

func (ts *nsmBulkRegServer) Send(reg *registry.NSERegistration) error {
	if ts.n.before != nil {
		var err error
		reg, err = ts.n.before(ts.ctx, reg)
		if err != nil {
			return err
		}
	}
	err := ts.server.Send(reg)
	return err
}
func (ts *nsmBulkRegServer) Recv() (*registry.NSERegistration, error) {
	reg, err := ts.server.Recv()
	if err != nil {
		return nil, err
	}
	if ts.n.after != nil {
		reg, err = ts.n.after(ts.ctx, reg)
		if err != nil {
			return nil, err
		}
	}
	return reg, err
}

func (ts *nsmBulkRegServer) Context() context.Context {
	return ts.ctx
}

// BulkRegisterNSE - register a stream of NSEs
func (n *customRegServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	ctx := server.Context()
	brs := &nsmBulkRegServer{
		ctx:    server.Context(),
		server: server,
		n:      n,
	}
	return next.NetworkServiceRegistryServer(ctx).BulkRegisterNSE(brs)
}

// RemoveNSE - remove NSE previously registered.
func (n *customRegServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.remove != nil {
		var err error
		request, err = n.remove(ctx, request)
		if err != nil {
			return nil, err
		}
	}
	return next.NetworkServiceRegistryServer(ctx).RemoveNSE(ctx, request)
}

// Compile level checks.
var _ registry.NetworkServiceRegistryServer = &customRegServer{}
