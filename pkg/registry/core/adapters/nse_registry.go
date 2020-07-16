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

// Package adapters provide API to converting client to server and vise versa
package adapters

import (
	"context"
	"errors"
	"io"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type networkServiceEndpointRegistryServer struct {
	client registry.NetworkServiceEndpointRegistryClient
}

func (n *networkServiceEndpointRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	doneCtx := withCapturedContext(ctx)
	nse, err := n.client.Register(doneCtx, request)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return nse, nil
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(lastCtx, request)
}

func (n *networkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	doneCtx := withCapturedContext(s.Context())
	client, err := n.client.Find(doneCtx, query)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}
	for {
		if err := client.Context().Err(); err != nil {
			break
		}
		msg, err := client.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		err = s.Send(msg)
		if err != nil {
			return err
		}
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx != nil {
		return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, streamcontext.NetworkServiceEndpointRegistryFindServer(lastCtx, s))
	}
	return nil
}

func (n *networkServiceEndpointRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	doneCtx := withCapturedContext(ctx)
	nse, err := n.client.Unregister(doneCtx, request)
	if err != nil {
		return nil, err
	}
	if request == nil {
		request = &registry.NetworkServiceEndpoint{}
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return nse, nil
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(lastCtx, request)
}

// NetworkServiceEndpointClientToServer - returns a registry.NetworkServiceEndpointRegistryClient wrapped around the supplied client
func NetworkServiceEndpointClientToServer(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryServer {
	return &networkServiceEndpointRegistryServer{client: next.NewNetworkServiceEndpointRegistryClient(client, &contextNSEClient{})}
}

var _ registry.NetworkServiceEndpointRegistryServer = &networkServiceEndpointRegistryServer{}

type networkServiceEndpointRegistryClient struct {
	server registry.NetworkServiceEndpointRegistryServer
}

func (n *networkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	doneCtx := withCapturedContext(ctx)
	nse, err := n.server.Register(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return nse, nil
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(lastCtx, in)
}

func (n *networkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEndpoint, channelSize)
	doneCtx := withCapturedContext(ctx)
	s := streamchannel.NewNetworkServiceEndpointFindServer(doneCtx, ch)
	if in != nil && in.Watch {
		go func() {
			defer close(ch)
			_ = n.server.Find(in, s)
		}()
	} else {
		defer close(ch)
		if err := n.server.Find(in, s); err != nil {
			return nil, err
		}
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx != nil {
		_, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(lastCtx, in, opts...)
		if err != nil {
			return nil, err
		}
	}
	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, ch), nil
}

func (n *networkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	doneCtx := withCapturedContext(ctx)
	nse, err := n.server.Unregister(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return nse, nil
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(lastCtx, in)
}

var _ registry.NetworkServiceEndpointRegistryClient = &networkServiceEndpointRegistryClient{}

// NetworkServiceEndpointServerToClient - returns a registry.NetworkServiceEndpointRegistryServer wrapped around the supplied server
func NetworkServiceEndpointServerToClient(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryClient {
	return &networkServiceEndpointRegistryClient{server: next.NewNetworkServiceEndpointRegistryServer(server, &contextNSEServer{})}
}
