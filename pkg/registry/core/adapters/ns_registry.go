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

package adapters

import (
	"context"
	"errors"
	"io"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"google.golang.org/grpc"

	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type networkServiceRegistryServer struct {
	client registry.NetworkServiceRegistryClient
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	doneCtx := withCapturedContext(ctx)
	ns, err := n.client.Register(doneCtx, request)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return ns, nil
	}
	return next.NetworkServiceRegistryServer(ctx).Register(lastCtx, request)
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
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
		return next.NetworkServiceRegistryServer(s.Context()).Find(query, streamcontext.NetworkServiceRegistryFindServer(lastCtx, s))
	}
	return nil
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	doneCtx := withCapturedContext(ctx)
	ns, err := n.client.Unregister(doneCtx, request)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return ns, nil
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(lastCtx, request)
}

// NetworkServiceClientToServer - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client
func NetworkServiceClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{client: next.NewNetworkServiceRegistryClient(client, &contextNSClient{})}
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}

type networkServiceRegistryClient struct {
	server registry.NetworkServiceRegistryServer
}

func (n *networkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*registry.NetworkService, error) {
	doneCtx := withCapturedContext(ctx)
	ns, err := n.server.Register(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return ns, nil
	}
	return next.NetworkServiceRegistryClient(ctx).Register(lastCtx, in)
}

func (n *networkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkService, channelSize)
	doneCtx := withCapturedContext(ctx)
	s := streamchannel.NewNetworkServiceFindServer(doneCtx, ch)
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
		_, err := next.NetworkServiceRegistryClient(ctx).Find(lastCtx, in)
		if err != nil {
			return nil, err
		}
	}
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}

func (n *networkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	doneCtx := withCapturedContext(ctx)
	ns, err := n.server.Unregister(doneCtx, in)
	if err != nil {
		return nil, err
	}
	lastCtx := getCapturedContext(doneCtx)
	if lastCtx == nil {
		return ns, nil
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(lastCtx, in)
}

var _ registry.NetworkServiceRegistryClient = &networkServiceRegistryClient{}

// NetworkServiceServerToClient - returns a registry.NetworkServiceRegistryServer wrapped around the supplied server
func NetworkServiceServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &networkServiceRegistryClient{server: next.NewNetworkServiceRegistryServer(server, &contextNSServer{})}
}
