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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type serializeNSEClient struct {
	executor multiexecutor.MultiExecutor
}

// NewNetworkServiceEndpointRegistryClient returns a new serialize NSE registry client chain element
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return new(serializeNSEClient)
}

func (c *serializeNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (reg *registry.NetworkServiceEndpoint, err error) {
	<-c.executor.AsyncExec(nse.Name, func() {
		registerCtx := serializectx.WithMultiExecutor(ctx, &c.executor)
		reg, err = next.NetworkServiceEndpointRegistryClient(ctx).Register(registerCtx, nse, opts...)
	})
	return reg, err
}

func (c *serializeNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *serializeNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	<-c.executor.AsyncExec(nse.Name, func() {
		_, err = next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
	})
	return new(empty.Empty), err
}
