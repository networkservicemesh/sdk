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

// Package setlogoption implements a chain element to set log options before full chain
package setlogoption

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type setNSLogOption struct {
	options map[string]string
	server  registry.NetworkServiceRegistryServer
}

type setNSLogOptionFindServer struct {
	registry.NetworkServiceRegistry_FindServer
	ctx context.Context
}

func (s *setNSLogOptionFindServer) Send(service *registry.NetworkService) error {
	return s.NetworkServiceRegistry_FindServer.Send(service)
}

func (s *setNSLogOptionFindServer) Context() context.Context {
	return s.ctx
}

func (s *setNSLogOption) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = s.withFields(ctx)
	return s.server.Register(ctx, service)
}

func (s *setNSLogOption) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := s.withFields(server.Context())
	return s.server.Find(query, &setNSLogOptionFindServer{ctx: ctx, NetworkServiceRegistry_FindServer: server})
}

func (s *setNSLogOption) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	ctx = s.withFields(ctx)
	return s.server.Unregister(ctx, service)
}

// NewNetworkServiceRegistryServer creates new instance of NewNetworkServiceRegistryServer which sets the passed options
func NewNetworkServiceRegistryServer(options map[string]string, server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &setNSLogOption{
		options: options,
		server:  server,
	}
}

func (s *setNSLogOption) withFields(ctx context.Context) context.Context {
	fields := make(logrus.Fields)
	for k, v := range s.options {
		fields[k] = v
	}
	if len(fields) > 0 {
		ctx = log.WithFields(ctx, fields)
	}
	return ctx
}
