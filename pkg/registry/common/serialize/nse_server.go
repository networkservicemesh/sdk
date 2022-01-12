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

type serializeNSEServer struct {
	executor multiexecutor.MultiExecutor
}

// NewNetworkServiceEndpointRegistryServer returns a new serialize NSE registry server chain element
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(serializeNSEServer)
}

func (s *serializeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (reg *registry.NetworkServiceEndpoint, err error) {
	<-s.executor.AsyncExec(nse.Name, func() {
		registerCtx := serializectx.WithMultiExecutor(ctx, &s.executor)
		reg, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(registerCtx, nse)
	})
	return reg, err
}

func (s *serializeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *serializeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (_ *empty.Empty, err error) {
	<-s.executor.AsyncExec(nse.Name, func() {
		_, err = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	})
	return new(empty.Empty), err
}
