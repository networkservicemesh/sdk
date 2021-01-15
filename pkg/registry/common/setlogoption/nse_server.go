// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package setlogoption implements a chain element to set log options before full chain
package setlogoption

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setNSELogOption struct {
	options map[string]string
}

type setNSELogOptionFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	ctx context.Context
}

func (s *setNSELogOptionFindServer) Send(endpoint *registry.NetworkServiceEndpoint) error {
	return s.NetworkServiceEndpointRegistry_FindServer.Send(endpoint)
}

func (s *setNSELogOptionFindServer) Context() context.Context {
	return s.ctx
}

func (s *setNSELogOption) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = s.withFields(ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (s *setNSELogOption) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := s.withFields(server.Context())
	return next.NetworkServiceEndpointRegistryServer(ctx).Find(query, &setNSELogOptionFindServer{ctx: ctx, NetworkServiceEndpointRegistry_FindServer: server})
}

func (s *setNSELogOption) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ctx = s.withFields(ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceEndpointRegistryServer which sets the passed options
func NewNetworkServiceEndpointRegistryServer(options map[string]string) registry.NetworkServiceEndpointRegistryServer {
	return &setNSELogOption{
		options: options,
	}
}

func (s *setNSELogOption) withFields(ctx context.Context) context.Context {
	fields := make(map[string]interface{})
	for k, v := range s.options {
		fields[k] = v
	}
	if len(fields) > 0 {
		ctx = logger.WithFields(ctx, fields)
	}
	return ctx
}
