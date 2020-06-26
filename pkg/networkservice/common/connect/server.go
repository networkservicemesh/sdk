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

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Contain grpc connection and client connection associated
type clientEntry struct {
	clientURI       *url.URL
	client          networkservice.NetworkServiceClient
	cc              grpc.ClientConnInterface
	ready           chan struct{} // Flag for connection is establishing
	connections     map[string]*networkservice.Connection
	err             error
	closeConnection func() error
}

func (ce *clientEntry) markAsReady() {
	close(ce.ready)
}

type connectServer struct {
	ctx           context.Context
	dialOptions   []grpc.DialOption
	clientFactory func(ctx context.Context, conn grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clients       map[string]*clientEntry // key == url as string
	connections   map[string]*url.URL     // Connection map is required to close using connection id.
	executor      serialize.Executor
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
	return &connectServer{
		ctx:           nil,
		clientFactory: clientFactory,
		clients:       map[string]*clientEntry{},
		connections:   map[string]*url.URL{},
		dialOptions:   clientDialOptions,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clientURL, err := c.clientURL(ctx, request.Connection)
	if err != nil {
		return nil, err
	}
	// Find or Create clientEntry, with add pending.
	var ce *clientEntry
	ce, err = c.findOrCreateClient(ctx, clientURL, request.Connection)
	if err != nil {
		// In case of error, no pending state is propogated.
		return nil, err
	}

	// We have connection ready,
	var conn *networkservice.Connection
	conn, err = ce.client.Request(ctx, request)
	if err != nil {
		// We not succeed, just remove connection by id, if it was existing one.
		<-c.executor.AsyncExec(func() {
			if request.Connection.Id != "" {
				// Delete from CE
				delete(ce.connections, request.Connection.Id)
				// Delete from global list
				delete(c.connections, request.Connection.Id)
			}
		})
		return nil, err
	}

	// Copy conn to request and pass to next one
	if conn.GetContext() != nil {
		request.Connection.Context = conn.GetContext()
	}
	// we succeed, update connection and remove pending state
	<-c.executor.AsyncExec(func() {
		ce.connections[conn.Id] = conn

		// Also update global connection map
		c.connections[conn.Id] = clientURL
	})

	return next.Server(ctx).Request(ctx, request)
}

func (c *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	clientURL, err := c.clientURL(ctx, conn)
	if err != nil {
		return nil, err
	}
	// Find or Create clientEntry, with add pending.
	var ce *clientEntry
	ce, err = c.findOrCreateClient(ctx, clientURL, conn)
	if err != nil {
		// In case of error, no pending state is propogated.
		return nil, err
	}
	// Call next Close before calling client Close
	_, err = next.Server(ctx).Close(ctx, conn)

	// Call client close
	if _, closeErr := ce.client.Close(ctx, conn); closeErr != nil {
		if err != nil {
			// Combine errors
			err = errors.Wrapf(err, "errors during close: %v", closeErr)
		}
	}

	// remove connection from list
	<-c.executor.AsyncExec(func() {
		delete(ce.connections, conn.Id)
		delete(c.connections, conn.Id)
		// Close client if there is no more users for it.
		c.closeClient(ctx, ce)
	})

	return &empty.Empty{}, err
}

func (c *connectServer) findOrCreateClient(ctx context.Context, clientURI *url.URL, conn *networkservice.Connection) (ce *clientEntry, err error) {
	connectionExists := false
	<-c.executor.AsyncExec(func() {
		ce, connectionExists = c.clients[clientURI.String()]
		if !connectionExists {
			ce = &clientEntry{
				clientURI:   clientURI,
				ready:       make(chan struct{}),
				connections: map[string]*networkservice.Connection{},
			}
			// Put/Update connection
			c.clients[clientURI.String()] = ce
		}
		ce.connections[conn.Id] = conn
	})

	if connectionExists {
		// Wait until connection is ready, or context timeout
		select {
		case <-ctx.Done():
			// We failed, we need to remove pending state and close connection if we are last one.
			<-c.executor.AsyncExec(func() {
				delete(ce.connections, conn.Id)
				c.closeClient(ctx, ce)
			})
			return nil, errors.Errorf("context timeout")
		case <-ce.ready:
			// all is fine, just return
			if ce.err != nil {
				<-c.executor.AsyncExec(func() {
					delete(ce.connections, conn.Id)
					c.closeClient(ctx, ce)
				})
				return nil, ce.err
			}
			return ce, nil
		}
	}
	// Dial and create client connection
	if err := c.createClient(ce); err != nil {
		// If we failed to create client, we failed to dial, so no need to close,
		// we need to mark it as error one and all clients pending will return errors, and last one will remove entry
		// Mark connection as ready to use
		<-c.executor.AsyncExec(func() {
			ce.err = err
			// Mark as ready but with error
			ce.markAsReady()
			c.closeClient(ctx, ce)
		})
		return nil, err
	}

	// Mark connection as ready to use
	ce.markAsReady()

	return ce, nil
}

// Should be called insice executor
func (c *connectServer) closeClient(ctx context.Context, ce *clientEntry) {
	if len(ce.connections) == 0 {
		if ce.cc != nil {
			// Close connection
			if err := ce.closeConnection(); err != nil {
				trace.Log(ctx).Errorf("error closing connection %v", err)
			}
		}
		delete(c.clients, ce.clientURI.String())
	}
}

func (c *connectServer) createClient(ce *clientEntry) (err error) {
	// Opening GPRC connection
	// we should open a connecton with specified server, and we should be sure we do this once.
	clientCtx, _ := context.WithCancel(context.Background())

	dialOptions := c.dialOptions

	// Dial with connection
	ce.cc, ce.closeConnection, err = grpcDialer(clientCtx, grpcutils.URLToTarget(ce.clientURI), dialOptions...)
	if err != nil {
		return errors.Wrapf(err, "unable to dial %s", ce.clientURI.String())
	}

	// Initialize client and factory
	ce.client = c.clientFactory(clientCtx, ce.cc)
	return nil
}

func (c *connectServer) clientURL(ctx context.Context, connection *networkservice.Connection) (*url.URL, error) {
	clientURL := clienturl.ClientURL(ctx)

	if clientURL == nil {
		<-c.executor.AsyncExec(func() {
			clientURL = c.connections[connection.Id]
		})
	}
	if clientURL == nil {
		return nil, errors.Errorf("a proper clienturl.ClientURL should be passed or request.Connection should be active. Connection: %v", connection)
	}
	return clientURL, nil
}
func grpcDialer(ctx context.Context, target string, opts ...grpc.DialOption) (grpc.ClientConnInterface, func() error, error) {
	grpcCon, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, nil, err
	}
	return grpcCon, grpcCon.Close, err
}
