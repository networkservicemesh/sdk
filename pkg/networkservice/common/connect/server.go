// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
)

type connectServer struct {
	ctx               context.Context
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clientDialOptions []grpc.DialOption
	clientsByURL      clientmap.RefcountMap // key == clientURL.String()
	clientsByID       clientmap.Map         // key == client connection ID
}

// NewServer - chain element that
func NewServer(ctx context.Context, clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient, clientDialOptions ...grpc.DialOption) networkservice.NetworkServiceServer {
	return &connectServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialOptions: clientDialOptions,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clientConn, clientErr := c.client(ctx, request.GetConnection()).Request(ctx, request)

	if clientErr != nil {
		return nil, clientErr
	}
	// Copy Context from client to response from server
	request.GetConnection().Context = clientConn.Context
	request.GetConnection().Path = clientConn.Path

	// Carry on with next.Server
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Return result
	return conn, err
}

func (c *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	client, _ := c.clientsByID.Load(conn.GetId())
	if client == nil {
		return &empty.Empty{}, nil
	}
	_, clientErr = client.Close(ctx, conn)
	rv, err := next.Server(ctx).Close(ctx, conn)
	if clientErr != nil && err != nil {
		return rv, errors.Wrapf(err, "errors during client close: %v", clientErr)
	}
	if clientErr != nil {
		return rv, errors.Wrap(clientErr, "errors during client close")
	}
	return rv, err
}

func (c *connectServer) client(ctx context.Context, conn *networkservice.Connection) networkservice.NetworkServiceClient {
	// Fast path, if we have a client for this conn.GetId(), use it
	client, _ := c.clientsByID.Load(conn.GetId())

	// If we didn't find a client, we fall back to clientURL
	if client == nil {
		clientURL := clienturl.ClientURL(ctx)
		// If we don't have a clientURL, all we can do is return errors
		if clientURL == nil {
			clientErr := errors.Errorf("clientURL not found for incoming connection: %+v", conn)
			return injecterror.NewClient(clientErr)
		}
		// Fast path: load the client by URL.  In the unfortunate event someone has poorly chosen
		// a clientFactory or dialOptions that take time, this will be faster than
		// creating a new one and doing a LoadOrStore
		client, _ = c.clientsByURL.Load(clientURL.String())
		// If we still don't have a client, create one, and LoadOrStore it
		// Note: It is possible for multiple nearly simultaneous initial Requests to race this client == nil check
		// and both enter into the body of the if block.  However, if that occurs, when they call call clientByID.LoadAndStore
		// only one of them will Store, the rest will Load.
		// The return of load == true from the clientById.LoadAndStore(conn.GetId()) indicates that there has been
		// a race, and those receiving it must then call clientByURL.Delete(clientURL) to clean up the Refcount, so
		// we only get one refcount increment per conn.GetId().  In this way correctness is preserved, even in the
		// unlikely case of multiple nearly simultaneous initial Requests racing this client == nil
		if client == nil {
			// Note: clienturl.NewClient(...) will get properly cleaned up when dereferences
			client = clienturl.NewClient(clienturl.WithClientURL(c.ctx, clientURL), c.clientFactory, c.clientDialOptions...)
			client, _ = c.clientsByURL.LoadOrStore(clientURL.String(), client)
			// Wrap the client in a per-connection connect.NewClient(...)
			// when this client receive a 'Close' it will call the cancelFunc provided deleting it from the various
			// maps.
			client = chain.NewNetworkServiceClient(
				NewClient(func() {
					c.clientsByID.Delete(conn.GetId())
					c.clientsByURL.Delete(clientURL.String())
				}),
				client,
			)
			var loaded bool
			client, loaded = c.clientsByID.LoadOrStore(conn.GetId(), client)
			if loaded {
				// If loaded == true, then another Request for the same conn.GetId() was being processed in parallel
				// since both of those called c.clientsByURL.LoadOrStore, the refcount for this one conn.GetId()
				// got incremented *twice*.  Correct for that here by decrementing
				c.clientsByURL.Delete(conn.GetId())
			}
		}
	}
	return client
}
