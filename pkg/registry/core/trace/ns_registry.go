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

// Package trace provides wrappers for tracing around a registry.{Registry,Discovery}{Server,Client}
package trace

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/trace/traceconcise"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace/traceverbose"
)

type traceNetworkServiceRegistryClient struct {
	verbose registry.NetworkServiceRegistryClient
	concise registry.NetworkServiceRegistryClient
}

// NewNetworkServiceRegistryClient - wraps registry.NetworkServiceRegistryClient with tracing.
func NewNetworkServiceRegistryClient(traced registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &traceNetworkServiceRegistryClient{
		verbose: traceverbose.NewNetworkServiceRegistryClient(traced),
		concise: traceconcise.NewNetworkServiceRegistryClient(traced),
	}
}

func (t *traceNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Register(ctx, in, opts...)
	}
	return t.concise.Register(ctx, in, opts...)
}

func (t *traceNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Find(ctx, in, opts...)
	}
	return t.concise.Find(ctx, in, opts...)
}

func (t *traceNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Unregister(ctx, in, opts...)
	}
	return t.concise.Unregister(ctx, in, opts...)
}

type traceNetworkServiceRegistryServer struct {
	verbose registry.NetworkServiceRegistryServer
	concise registry.NetworkServiceRegistryServer
}

// NewNetworkServiceRegistryServer - wraps registry.NetworkServiceRegistryServer with tracing.
func NewNetworkServiceRegistryServer(traced registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &traceNetworkServiceRegistryServer{
		verbose: traceverbose.NewNetworkServiceRegistryServer(traced),
		concise: traceconcise.NewNetworkServiceRegistryServer(traced),
	}
}

func (t *traceNetworkServiceRegistryServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Register(ctx, in)
	}
	return t.concise.Register(ctx, in)
}

func (t *traceNetworkServiceRegistryServer) Find(in *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Find(in, s)
	}
	return t.concise.Find(in, s)
}

func (t *traceNetworkServiceRegistryServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Unregister(ctx, in)
	}
	return t.concise.Unregister(ctx, in)
}
