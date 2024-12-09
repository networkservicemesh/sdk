// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package traceverbose

import (
	"context"
	"io"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type traceNetworkServiceEndpointRegistryClient struct {
	traced registry.NetworkServiceEndpointRegistryClient
}

type traceNetworkServiceEndpointRegistryFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
	operation string
}

func (t *traceNetworkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	ctx, finish := withLog(t.Context(), t.operation, methodNameRecv)
	defer finish()

	s := streamcontext.NetworkServiceEndpointRegistryFindClient(ctx, t.NetworkServiceEndpointRegistry_FindClient)
	rv, err := s.Recv()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.WithStack(err)
		}
		if status.Code(err) == codes.Canceled {
			return nil, errors.WithStack(err)
		}
		return nil, logError(ctx, err, t.operation)
	}
	logObjectTrace(ctx, "recv-response", rv)
	return rv, nil
}

func (t *traceNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.traced, methodNameRegister)
	ctx, finish := withLog(ctx, operation, methodNameRegister)
	defer finish()

	logObjectTrace(ctx, "register", in)
	rv, err := t.traced.Register(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "register-response", rv)
	return rv, nil
}
func (t *traceNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	operation := typeutils.GetFuncName(t.traced, methodNameFind)
	ctx, finish := withLog(ctx, operation, methodNameFind)
	defer finish()

	logObjectTrace(ctx, "find", in)

	// Actually call the next
	rv, err := t.traced.Find(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "find-response", rv)

	return &traceNetworkServiceEndpointRegistryFindClient{NetworkServiceEndpointRegistry_FindClient: rv, operation: typeutils.GetFuncName(t.traced, methodNameRecv)}, nil
}

func (t *traceNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
	ctx, finish := withLog(ctx, operation, methodNameUnregister)
	defer finish()

	logObjectTrace(ctx, "unregister", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "unregister-response", rv)
	return rv, nil
}

// NewNetworkServiceEndpointRegistryClient - wraps registry.NetworkServiceEndpointRegistryClient with tracing
func NewNetworkServiceEndpointRegistryClient(traced registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &traceNetworkServiceEndpointRegistryClient{traced: traced}
}

type traceNetworkServiceEndpointRegistryServer struct {
	traced registry.NetworkServiceEndpointRegistryServer
}

func (t *traceNetworkServiceEndpointRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.traced, methodNameRegister)
	ctx, finish := withLog(ctx, operation, methodNameRegister)
	defer finish()

	logObjectTrace(ctx, "register", in)

	rv, err := t.traced.Register(ctx, in)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "register-response", rv)
	return rv, nil
}

func (t *traceNetworkServiceEndpointRegistryServer) Find(in *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	operation := typeutils.GetFuncName(t.traced, methodNameFind)
	ctx, finish := withLog(s.Context(), operation, methodNameFind)
	defer finish()

	s = &traceNetworkServiceEndpointRegistryFindServer{
		NetworkServiceEndpointRegistry_FindServer: streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s),
		operation: typeutils.GetFuncName(t.traced, methodNameSend),
	}
	logObjectTrace(ctx, "find", in)

	// Actually call the next
	err := t.traced.Find(in, s)
	if err != nil {
		return logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "find-response", in)

	return nil
}

func (t *traceNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
	ctx, finish := withLog(ctx, operation, methodNameUnregister)
	defer finish()

	logObjectTrace(ctx, "unregister", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in)
	if err != nil {
		return nil, logError(ctx, err, operation)
	}
	logObjectTrace(ctx, "unregister-response", in)
	return rv, nil
}

// NewNetworkServiceEndpointRegistryServer - wraps registry.NetworkServiceEndpointRegistryServer with tracing
func NewNetworkServiceEndpointRegistryServer(traced registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &traceNetworkServiceEndpointRegistryServer{traced: traced}
}

type traceNetworkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	operation string
}

func (t *traceNetworkServiceEndpointRegistryFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	ctx, finish := withLog(t.Context(), t.operation, methodNameSend)
	defer finish()

	logObjectTrace(ctx, "network service endpoint", nseResp.NetworkServiceEndpoint)
	s := streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, t.NetworkServiceEndpointRegistry_FindServer)
	err := s.Send(nseResp)
	if err != nil {
		return logError(ctx, err, t.operation)
	}
	return nil
}
