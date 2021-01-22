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

package setlogoption

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setNSLogOption struct {
	options map[string]string
}

type setLogOptionFindServer struct {
	registry.NetworkServiceRegistry_FindServer
	ctx context.Context
}

func (s *setLogOptionFindServer) Send(ns *registry.NetworkService) error {
	return s.NetworkServiceRegistry_FindServer.Send(ns)
}

func (s *setLogOptionFindServer) Context() context.Context {
	return s.ctx
}

func (s *setNSLogOption) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = s.withFields(ctx)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *setNSLogOption) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := s.withFields(server.Context())
	return next.NetworkServiceRegistryServer(ctx).Find(query, &setLogOptionFindServer{ctx: ctx, NetworkServiceRegistry_FindServer: server})
}

func (s *setNSLogOption) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	ctx = s.withFields(ctx)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

// NewNetworkServiceRegistryServer creates new instance of NetworkServiceRegistryServer which sets the passed options
func NewNetworkServiceRegistryServer(options map[string]string) registry.NetworkServiceRegistryServer {
	return &setNSLogOption{
		options: options,
	}
}

func (s *setNSLogOption) withFields(ctx context.Context) context.Context {
	fields := make(map[string]interface{})
	for k, v := range s.options {
		fields[k] = v
	}
	if len(fields) > 0 {
		ctx = logger.WithFields(ctx, fields)
	}
	return ctx
}
