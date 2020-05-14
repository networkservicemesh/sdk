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

package join

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"google.golang.org/grpc"
)

type nsmRegistryClient struct {
	clients []registry.NsmRegistryClient
}

func (n *nsmRegistryClient) RegisterNSM(ctx context.Context, in *registry.NetworkServiceManager, opts ...grpc.CallOption) (*registry.NetworkServiceManager, error) {
	for _, c := range n.clients {
		_, _ = c.RegisterNSM(ctx, in, opts...)
	}
	return next.NSMRegistryClient(ctx).RegisterNSM(ctx, in, opts...)
}

func (n *nsmRegistryClient) GetEndpoints(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*registry.NetworkServiceEndpointList, error) {
	var result = new(registry.NetworkServiceEndpointList)
	var set = map[string]*registry.NetworkServiceEndpoint{}
	for _, c := range n.clients {
		list, err := c.GetEndpoints(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		for _, nse := range list.NetworkServiceEndpoints {
			if _, ok := set[nse.Name]; ok {
				continue
			}
			set[nse.Name] = nse
			result.NetworkServiceEndpoints = append(result.NetworkServiceEndpoints, nse)
		}
	}
	return result, nil
}

func NewNsmRegistryClient(clients ...registry.NsmRegistryClient) registry.NsmRegistryClient {
	return &nsmRegistryClient{clients: clients}
}

var _ registry.NsmRegistryClient = &nsmRegistryClient{}
