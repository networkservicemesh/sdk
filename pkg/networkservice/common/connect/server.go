// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package connect is intended to allow passthrough style Endpoints to have a server that also connects to a client
package connect

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
)

type connectServer struct {
	ctx                     context.Context
	dialOptions             []grpc.DialOption
	clientFactory           func(ctx context.Context, conn grpc.ClientConnInterface) networkservice.NetworkServiceClient
	uRLToClientMap          clientMap // key == url as string
	connectionIDToClientMap clientMap // key == connection.GetId()
	dialOptionFactories     []func(ctx context.Context, request *networkservice.NetworkServiceRequest, clientURL *url.URL) []grpc.DialOption
}

type dialOptionFactory struct {
	factory func(ctx context.Context, request *networkservice.NetworkServiceRequest, clientURL *url.URL) []grpc.DialOption
	grpc.EmptyDialOption
}

// WithDialOptionFactory - define a dial option factory, will be called on moment to create dial
func WithDialOptionFactory(factory func(ctx context.Context, request *networkservice.NetworkServiceRequest, clientURL *url.URL) []grpc.DialOption) grpc.DialOption {
	return &dialOptionFactory{
		factory: factory,
	}
}

// NewServer - returns a new connect Server
//             clientFactory - a function which takes a ctx that governs the lifecycle of the client and
//                             a cc grpc.ClientConnInterface and returns a networkservice.NetworkServiceClient
//                             The returned client will be called with the same inputs that were passed to the connect Server.
//                             This means that the client returned by clientFactory is responsible for any mutations to that
//                             request (setting a new id, setting different Mechanism Preferences etc) and any mutations
//                             before returning to the server.
//             connect presumes depends on some previous chain element having set clienturl.WithClientURL so it can know
//             which client to address.
func NewServer(clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient, clientDialOptions ...grpc.DialOption) networkservice.NetworkServiceServer {
	dialOptionFactories := []func(ctx context.Context, request *networkservice.NetworkServiceRequest, clientURL *url.URL) []grpc.DialOption{}
	for _, op := range clientDialOptions {
		if f, ok := op.(*dialOptionFactory); ok {
			dialOptionFactories = append(dialOptionFactories, f.factory)
		}
	}
	return &connectServer{
		ctx:                     nil,
		clientFactory:           clientFactory,
		uRLToClientMap:          newClientMap(),
		connectionIDToClientMap: newClientMap(),
		dialOptions:             clientDialOptions,
		dialOptionFactories:     dialOptionFactories,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	client, found := c.connectionIDToClientMap.Load(request.GetConnection().GetId())
	if found {
		return client.Request(ctx, request)
	}
	u := clienturl.ClientURL(ctx)
	client, found = c.uRLToClientMap.Load(u.String())
	if found {
		c.connectionIDToClientMap.LoadOrStore(request.GetConnection().GetId(), client)
		return client.Request(ctx, request)
	}
	// TODO - do something with cancelFunc to clean up uRLToClientMap when we are done with them (maybe ref counting on Close?)
	clientCtx, _ := context.WithCancel(context.Background())
	// TODO - fix to accept dialOptions from github.com/networkservicemesh/sdk/tools/security - the options should flow in from the top
	// TODO - fix to accept a mockable interface we can pass in from the top other than DialContext
	// TODO - fix to be cautious about schemes
	dialOptions := c.dialOptions
	for _, dof := range c.dialOptionFactories {
		dialOptions = append(dialOptions, dof(ctx, request, u)...)
	}
	cc, err := grpc.DialContext(clientCtx, grpcutils.URLToTarget(u), dialOptions...)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to dial %s", u.String())
	}
	client = c.clientFactory(clientCtx, cc)
	client, _ = c.uRLToClientMap.LoadOrStore(u.String(), client)
	client, _ = c.connectionIDToClientMap.LoadOrStore(request.GetConnection().GetId(), client)
	conn, err := client.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func (c *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	client, found := c.connectionIDToClientMap.Load(conn.GetId())
	if found {
		return client.Close(ctx, conn)
	}
	u := clienturl.ClientURL(ctx)
	client, found = c.uRLToClientMap.Load(u.String())
	if found {
		c.connectionIDToClientMap.LoadOrStore(conn.GetId(), client)
		return client.Close(ctx, conn)
	}
	// TODO - do something with cancelFunc to clean up uRLToClientMap when we are done with them (maybe ref counting on Close?)
	clientCtx, _ := context.WithCancel(context.Background())
	// TODO - fix to accept dialOptions from github.com/networkservicemesh/sdk/tools/security - the options should flow in from the top
	// TODO - fix to accept a mockable interface we can pass in from the top other than DialContext
	// TODO - fix to be cautious about schemes
	cc, err := grpc.DialContext(clientCtx, u.String(), c.dialOptions...)
	if err != nil {
		return nil, err
	}
	client = c.clientFactory(clientCtx, cc)
	client, _ = c.uRLToClientMap.LoadOrStore(u.String(), client)
	client, _ = c.connectionIDToClientMap.LoadOrStore(conn.GetId(), client)
	return client.Close(ctx, conn)
}
