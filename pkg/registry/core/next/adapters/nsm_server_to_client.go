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

package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsmServerToClient struct {
	server registry.NsmRegistryServer
	next   func(ctx context.Context) registry.NsmRegistryClient
}

// NewNSMServerToClient - returns a registry.NsmRegistryClient wrapped around the supplied server
func NewNSMServerToClient(client registry.NsmRegistryServer, next func(ctx context.Context) registry.NsmRegistryClient) registry.NsmRegistryClient {
	return &nsmServerToClient{server: client, next: next}
}

func (n *nsmServerToClient) RegisterNSM(ctx context.Context, in *registry.NetworkServiceManager, opts ...grpc.CallOption) (*registry.NetworkServiceManager, error) {
	result, err := n.server.RegisterNSM(ctx, in)
	if err != nil || n.next == nil {
		return result, err
	}
	return n.next(ctx).RegisterNSM(ctx, in, opts...)
}

func (n *nsmServerToClient) GetEndpoints(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*registry.NetworkServiceEndpointList, error) {
	result, err := n.server.GetEndpoints(ctx, in)
	if err != nil || n.next == nil {
		return nil, err
	}
	var nextResult *registry.NetworkServiceEndpointList
	nextResult, err = n.next(ctx).GetEndpoints(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	result.NetworkServiceEndpoints = append(result.NetworkServiceEndpoints, nextResult.NetworkServiceEndpoints...)
	return result, nil
}

// Implementation check
var _ registry.NsmRegistryClient = &nsmServerToClient{}
