// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

// Package trace provides a wrapper for tracing around a registry.{Registry,Discovery}{Server,Client}
package trace

import (
	"context"
	"errors"
	"io"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type traceNetworkServiceRegistryClient struct {
	traced registry.NetworkServiceRegistryClient
}

type traceNetworkServiceRegistryFindClient struct {
	registry.NetworkServiceRegistry_FindClient
}

func (t *traceNetworkServiceRegistryFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	operation := typeutils.GetFuncName(t.NetworkServiceRegistry_FindClient, "Recv")
	ctx, finish := withLog(t.Context(), operation)
	defer finish()

	s := streamcontext.NetworkServiceRegistryFindClient(ctx, t.NetworkServiceRegistry_FindClient)
	rv, err := s.Recv()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		if status.Code(err) == codes.Canceled {
			return nil, err
		}
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "recv-response", rv.NetworkService)
	return rv, err
}

func (t *traceNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	operation := typeutils.GetFuncName(t.traced, "Register")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logObjectTrace(ctx, "register", in)

	rv, err := t.traced.Register(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "register-response", rv)
	return rv, err
}
func (t *traceNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	operation := typeutils.GetFuncName(t.traced, "Find")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logObjectTrace(ctx, "find", in)

	// Actually call the next
	rv, err := t.traced.Find(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "find-response", rv)

	return &traceNetworkServiceRegistryFindClient{NetworkServiceRegistry_FindClient: rv}, nil
}

func (t *traceNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, "Unregister")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logObjectTrace(ctx, "unregister", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "unregister-response", rv)
	return rv, err
}

// NewNetworkServiceRegistryClient - wraps registry.NetworkServiceRegistryClient with tracing
func NewNetworkServiceRegistryClient(traced registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &traceNetworkServiceRegistryClient{traced: traced}
}

type traceNetworkServiceRegistryServer struct {
	traced registry.NetworkServiceRegistryServer
}

func (t *traceNetworkServiceRegistryServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	operation := typeutils.GetFuncName(t.traced, "Register")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logObjectTrace(ctx, "register", in)

	rv, err := t.traced.Register(ctx, in)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "register-response", rv)
	return rv, err
}

func (t *traceNetworkServiceRegistryServer) Find(in *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	operation := typeutils.GetFuncName(t.traced, "Find")
	ctx, finish := withLog(s.Context(), operation)
	defer finish()

	s = &traceNetworkServiceRegistryFindServer{
		NetworkServiceRegistry_FindServer: streamcontext.NetworkServiceRegistryFindServer(ctx, s),
	}
	logObjectTrace(ctx, "find", in)

	// Actually call the next
	err := t.traced.Find(in, s)
	if err != nil {
		return logError(ctx, err, operation)
	}
	return nil
}

func (t *traceNetworkServiceRegistryServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, "Unregister")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logObjectTrace(ctx, "unregister", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "unregister-response", rv)
	return rv, err
}

// NewNetworkServiceRegistryServer - wraps registry.NetworkServiceRegistryServer with tracing
func NewNetworkServiceRegistryServer(traced registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &traceNetworkServiceRegistryServer{traced: traced}
}

type traceNetworkServiceRegistryFindServer struct {
	registry.NetworkServiceRegistry_FindServer
}

func (t *traceNetworkServiceRegistryFindServer) Send(nsResp *registry.NetworkServiceResponse) error {
	operation := typeutils.GetFuncName(t.NetworkServiceRegistry_FindServer, "Send")
	ctx, finish := withLog(t.Context(), operation)
	defer finish()

	logObjectTrace(ctx, "network service", nsResp.NetworkService)

	s := streamcontext.NetworkServiceRegistryFindServer(ctx, t.NetworkServiceRegistry_FindServer)
	err := s.Send(nsResp)
	if err != nil {
		return logError(ctx, err, operation)
	}
	return err
}
