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

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	nsRegistryClient = "NetworkServiceRegistryClient"
)

type setNSLogOptionClient struct {
	options map[string]string
}

func (s *setNSLogOptionClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx = withFields(ctx, s.options, nsRegistryClient)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
}

func (s *setNSLogOptionClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ctx = withFields(ctx, s.options, nsRegistryClient)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *setNSLogOptionClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = withFields(ctx, s.options, nsRegistryClient)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}

// NewNetworkServiceRegistryClient creates new instance of NetworkServiceRegistryClient which sets the passed options
func NewNetworkServiceRegistryClient(options map[string]string) registry.NetworkServiceRegistryClient {
	return &setNSLogOptionClient{
		options: options,
	}
}
