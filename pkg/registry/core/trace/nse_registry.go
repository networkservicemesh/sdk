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

package trace

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/trace/traceconcise"
	"github.com/networkservicemesh/sdk/pkg/registry/core/trace/traceverbose"
)

type traceNetworkServiceEndpointRegistryClient struct {
	verbose registry.NetworkServiceEndpointRegistryClient
	concise registry.NetworkServiceEndpointRegistryClient
}

// NewNetworkServiceEndpointRegistryClient - wraps registry.NetworkServiceEndpointRegistryClient with tracing
func NewNetworkServiceEndpointRegistryClient(traced registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &traceNetworkServiceEndpointRegistryClient{
		verbose: traceverbose.NewNetworkServiceEndpointRegistryClient(traced),
		concise: traced,
	}
}

func (t *traceNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Register(ctx, in, opts...)
	}
	return t.concise.Register(ctx, in, opts...)
}
func (t *traceNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Find(ctx, in, opts...)
	}
	return t.concise.Find(ctx, in, opts...)
}

func (t *traceNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Unregister(ctx, in, opts...)
	}
	return t.concise.Unregister(ctx, in, opts...)
}

type traceNetworkServiceEndpointRegistryServer struct {
	verbose registry.NetworkServiceEndpointRegistryServer
	concise registry.NetworkServiceEndpointRegistryServer
}

// NewNetworkServiceEndpointRegistryServer - wraps registry.NetworkServiceEndpointRegistryServer with tracing
func NewNetworkServiceEndpointRegistryServer(traced registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &traceNetworkServiceEndpointRegistryServer{
		verbose: traceverbose.NewNetworkServiceEndpointRegistryServer(traced),
		concise: traceconcise.NewNetworkServiceEndpointRegistryServer(traced),
	}
}

func (t *traceNetworkServiceEndpointRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Register(ctx, in)
	}
	return t.concise.Register(ctx, in)
}

func (t *traceNetworkServiceEndpointRegistryServer) Find(in *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Find(in, s)
	}
	return t.concise.Find(in, s)
}

func (t *traceNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if logrus.GetLevel() == logrus.TraceLevel {
		return t.verbose.Unregister(ctx, in)
	}
	return t.concise.Unregister(ctx, in)
}
