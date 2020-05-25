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

// Package adapters provides adapters to translate between registry.{Registry,Discover}{Server,Client}
package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type registryServerToClient struct {
	server registry.NetworkServiceRegistryServer
}

type regErr struct {
	reg *registry.NSERegistration
	err error
}

type streamClient struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEClient
	sendCh chan *regErr
	recvCh chan *regErr
	err    error
}

type streamServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	sendCh chan *regErr
	recvCh chan *regErr
	ctx    context.Context
}

func (ss *streamServer) Context() context.Context {
	return ss.ctx
}

func (ts *streamClient) Send(reg *registry.NSERegistration) error {
	ts.sendCh <- &regErr{
		reg: reg,
	}
	return ts.err
}
func (ts *streamClient) Recv() (*registry.NSERegistration, error) {
	if ts.err != nil {
		return nil, ts.err
	}
	res := <-ts.recvCh
	return res.reg, res.err
}

func (ss *streamServer) Send(reg *registry.NSERegistration) error {
	ss.sendCh <- &regErr{
		reg: reg,
	}
	return nil
}
func (ss *streamServer) Recv() (*registry.NSERegistration, error) {
	res := <-ss.recvCh
	return res.reg, res.err
}

// NewRegistryServerToClient - returns a new registry.NetworkServiceRegistryServer that is a wrapper around server
func NewRegistryServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &registryServerToClient{server: server}
}

func (r *registryServerToClient) RegisterNSE(ctx context.Context, registration *registry.NSERegistration, _ ...grpc.CallOption) (*registry.NSERegistration, error) {
	return r.server.RegisterNSE(ctx, registration)
}

// BulkRegisterNSE - register in bulk
func (r *registryServerToClient) BulkRegisterNSE(ctx context.Context, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	recv := make(chan *regErr, 1)
	send := make(chan *regErr, 1)
	regClient := &streamClient{sendCh: send, recvCh: recv}
	// Reverse channels
	regServer := &streamServer{sendCh: recv, recvCh: send, ctx: ctx}
	regClient.err = r.server.BulkRegisterNSE(regServer)
	return regClient, regClient.err
}

func (r *registryServerToClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, _ ...grpc.CallOption) (*empty.Empty, error) {
	return r.server.RemoveNSE(ctx, request)
}
