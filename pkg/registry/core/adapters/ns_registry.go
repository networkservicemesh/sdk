// Copyright (c) 2020-2021 Cisco Systems, Inc.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package adapters provide API to converting client to server and vise versa
package adapters

import (
	"context"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type networkServiceRegistryServer struct {
	client registry.NetworkServiceRegistryClient
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	return next.NewNetworkServiceRegistryClient(
		n.client,
		&callNextNSServer{server: next.NetworkServiceRegistryServer(ctx)},
	).Register(ctx, request)
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	client, err := next.NewNetworkServiceRegistryClient(
		n.client,
		&callNextNSServer{server: next.NetworkServiceRegistryServer(server.Context())},
	).Find(server.Context(), query)
	if client == nil || err != nil {
		return err
	}
	return nsFindClientToServer(client, server)
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	return next.NewNetworkServiceRegistryClient(
		n.client,
		&callNextNSServer{server: next.NetworkServiceRegistryServer(ctx)},
	).Unregister(ctx, request)
}

// NetworkServiceClientToServer - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client.
func NetworkServiceClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{client: client}
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}

type networkServiceRegistryClient struct {
	server registry.NetworkServiceRegistryServer
}

func (n *networkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NewNetworkServiceRegistryServer(
		n.server,
		&callNextNSClient{client: next.NetworkServiceRegistryClient(ctx)},
	).Register(ctx, in)
}

func (n *networkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	s := next.NewNetworkServiceRegistryServer(
		n.server,
		&callNextNSClient{client: next.NetworkServiceRegistryClient(ctx)},
	)
	return nsFindServerToClient(ctx, s, in)
}

func (n *networkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	return next.NewNetworkServiceRegistryServer(
		n.server,
		&callNextNSClient{client: next.NetworkServiceRegistryClient(ctx)},
	).Unregister(ctx, in)
}

var _ registry.NetworkServiceRegistryClient = &networkServiceRegistryClient{}

// NetworkServiceServerToClient - returns a registry.NetworkServiceRegistryServer wrapped around the supplied server.
func NetworkServiceServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &networkServiceRegistryClient{server: server}
}

func nsFindClientToServer(client registry.NetworkServiceRegistry_FindClient, server registry.NetworkServiceRegistry_FindServer) error {
	for {
		if err := client.Context().Err(); err != nil {
			break
		}
		msg, err := client.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return errors.Wrap(err, "NetworkServiceRegistry find client failed to get a message")
		}
		err = server.Send(msg)
		if err != nil {
			return errors.Wrapf(err, "NetworkServiceRegistry find server failed to send a message %s", msg)
		}
	}
	return nil
}

func nsFindServerToClient(ctx context.Context, server registry.NetworkServiceRegistryServer, in *registry.NetworkServiceQuery) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceResponse, channelSize)
	s := streamchannel.NewNetworkServiceFindServer(ctx, ch)
	if in != nil && in.GetWatch() {
		go func() {
			defer close(ch)
			_ = server.Find(in, s)
		}()
	} else {
		defer close(ch)
		if err := server.Find(in, s); err != nil {
			return nil, errors.Wrap(err, "NetworkServiceRegistry find server failed to find a query")
		}
	}
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}
