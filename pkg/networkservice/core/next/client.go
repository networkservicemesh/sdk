// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type nextClient struct {
	clients    []networkservice.NetworkServiceClient
	index      int
	nextParent networkservice.NetworkServiceClient
}

// ClientWrapper - a function that wraps around a networkservice.NetworkServiceClient
type ClientWrapper func(networkservice.NetworkServiceClient) networkservice.NetworkServiceClient

// ClientChainer - a function that chains together a list of networkservice.NetworkServiceClients
type ClientChainer func(...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient

// NewWrappedNetworkServiceClient chains together clients with wrapper wrapped around each one
func NewWrappedNetworkServiceClient(wrapper ClientWrapper, clients ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	if len(clients) == 0 {
		return &tailClient{}
	}
	rv := &nextClient{clients: make([]networkservice.NetworkServiceClient, 0, len(clients))}
	for _, c := range clients {
		rv.clients = append(rv.clients, wrapper(c))
	}
	return rv
}

// NewNetworkServiceClient - chains together clients into a single networkservice.NetworkServiceClient
func NewNetworkServiceClient(clients ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return NewWrappedNetworkServiceClient(func(client networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
		return client
	}, clients...)
}

func (n *nextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if np := Client(ctx); np != nil {
			nextParent = np
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Request(withNextClient(ctx, &nextClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), request, opts...)
	}
	return n.clients[n.index].Request(withNextClient(ctx, nextParent), request, opts...)
}

func (n *nextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if np := Client(ctx); np != nil {
			nextParent = np
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Close(withNextClient(ctx, &nextClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), conn, opts...)
	}
	return n.clients[n.index].Close(withNextClient(ctx, nextParent), conn, opts...)
}
