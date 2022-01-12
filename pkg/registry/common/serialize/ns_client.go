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

type serializeNSClient struct {
	executor multiexecutor.MultiExecutor
}

// NewNetworkServiceRegistryClient returns a new serialize NS registry client chain element
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return new(serializeNSClient)
}

func (c *serializeNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (reg *registry.NetworkService, err error) {
	<-c.executor.AsyncExec(ns.Name, func() {
		registerCtx := serializectx.WithMultiExecutor(ctx, &c.executor)
		reg, err = next.NetworkServiceRegistryClient(ctx).Register(registerCtx, ns, opts...)
	})
	return reg, err
}

func (c *serializeNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *serializeNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	<-c.executor.AsyncExec(ns.Name, func() {
		_, err = next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
	})
	return new(empty.Empty), err
}
