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

package serialize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type serializeNSServer struct {
	executor multiexecutor.MultiExecutor
}

// NewNetworkServiceRegistryServer returns a new serialize NS registry server chain element
func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return new(serializeNSServer)
}

func (s *serializeNSServer) Register(ctx context.Context, ns *registry.NetworkService) (reg *registry.NetworkService, err error) {
	<-s.executor.AsyncExec(ns.Name, func() {
		registerCtx := serializectx.WithMultiExecutor(ctx, &s.executor)
		reg, err = next.NetworkServiceRegistryServer(ctx).Register(registerCtx, ns)
	})
	return reg, err
}

func (s *serializeNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *serializeNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (_ *empty.Empty, err error) {
	<-s.executor.AsyncExec(ns.Name, func() {
		_, err = next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	})
	return new(empty.Empty), err
}
