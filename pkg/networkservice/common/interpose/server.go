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

// Package interpose provides a NetworkServiceServer chain element that tracks local Cross connect Endpoints and call them first
// their unix file socket as the clienturl.ClientURL(ctx) used to connect to them.
package interpose

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type interposeServer struct {
	endpoints        stringurl.Map
	activeConnection connectionInfoMap // key == connectionId
}

type connectionInfo struct {
	clientConnID    string
	endpointURL     *url.URL
	interposeNSEURL *url.URL
}

// NewServer - creates a NetworkServiceServer that tracks locally registered CrossConnect Endpoints and on first Request forward to cross conenct nse
//				one by one and if request came back from cross nse, it will connect to a proper next client endpoint.
//             - server - *registry.NetworkServiceRegistryServer.  Since registry.NetworkServiceRegistryServer is an interface
//                        (and thus a pointer) *registry.NetworkServiceRegistryServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements registry.NetworkServiceRegistryServer
//                        This is done so that we can return a registry.NetworkServiceRegistryServer chain element
//                        while maintaining the NewServer pattern for use like anything else in a chain.
//                        The value in *server must be included in the registry.NetworkServiceRegistryServer listening
//                        so it can capture the registrations.
func NewServer(registryServer *registry.NetworkServiceEndpointRegistryServer) networkservice.NetworkServiceServer {
	rv := new(interposeServer)
	*registryServer = interpose.NewNetworkServiceRegistryServer(&rv.endpoints)
	return rv
}

func (l *interposeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (result *networkservice.Connection, err error) {
	// Check if there is no active connection, we need to replace endpoint url with forwarder url
	conn := request.GetConnection()
	ind := conn.GetPath().GetIndex() // It is designed to be used inside Endpoint, so current index is Endpoint already
	connID := conn.GetId()

	if len(conn.GetPath().GetPathSegments()) == 0 || ind <= 0 {
		return nil, errors.Errorf("path segment doesn't have a client or cross connect nse identity")
	}
	// We need to find an Id from path to match active connection request.
	activeConnID := l.getConnectionID(conn)

	// We came from client, so select cross nse and go to it.
	clientURL := clienturlctx.ClientURL(ctx)

	connInfo, ok := l.activeConnection.Load(activeConnID)
	if ok {
		if connID != activeConnID {
			l.activeConnection.Store(connID, connInfo)
		}
	} else {
		if connID != activeConnID {
			return nil, errors.Errorf("connection id should match current path segment id")
		}

		// Iterate over all cross connect NSEs to check one with passed state.
		l.endpoints.Range(func(key string, crossNSEURL *url.URL) bool {
			crossCTX := clienturlctx.WithClientURL(ctx, crossNSEURL)

			// Store client connection and selected cross connection URL.
			connInfo, _ = l.activeConnection.LoadOrStore(activeConnID, connectionInfo{
				clientConnID:    activeConnID,
				endpointURL:     clientURL,
				interposeNSEURL: crossNSEURL,
			})
			result, err = next.Server(crossCTX).Request(crossCTX, request)
			if err != nil {
				logger.Log(ctx).Errorf("failed to request cross NSE %v err: %v", crossNSEURL, err)
				return true
			}
			// If all is ok, stop iterating.
			return false
		})
		if result != nil {
			return result, nil
		}

		l.activeConnection.Delete(activeConnID)

		return nil, errors.Errorf("all cross NSE failed to connect to endpoint %v connection: %v", clientURL, conn)
	}

	var crossCTX context.Context
	if connID == connInfo.clientConnID {
		crossCTX = clienturlctx.WithClientURL(ctx, connInfo.interposeNSEURL)
	} else {
		// Go to endpoint URL if it matches one we had on previous step.
		if clientURL != connInfo.endpointURL && *clientURL != *connInfo.endpointURL {
			return nil, errors.Errorf("new selected endpoint URL %v doesn't match endpoint URL selected before interpose NSE %v", clientURL, connInfo.endpointURL)
		}
		crossCTX = ctx
	}

	return next.Server(crossCTX).Request(crossCTX, request)
}

func (l *interposeServer) getConnectionID(conn *networkservice.Connection) string {
	id := conn.Id
	for i := conn.GetPath().GetIndex(); i > 0; i-- {
		activeConnID := conn.GetPath().GetPathSegments()[i].Id
		if connInfo, ok := l.activeConnection.Load(activeConnID); ok {
			if activeConnID == connInfo.clientConnID {
				id = activeConnID
			}
			break
		}
	}
	return id
}

func (l *interposeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// If we came from NSMgr, we need to go to proper interpose NSE
	connInfo, ok := l.activeConnection.Load(conn.GetId())
	if !ok {
		return nil, errors.Errorf("no active connection found: %v", conn)
	}

	var crossCTX context.Context
	if conn.GetId() == connInfo.clientConnID {
		crossCTX = clienturlctx.WithClientURL(ctx, connInfo.interposeNSEURL)
	} else {
		crossCTX = ctx
	}

	l.activeConnection.Delete(conn.GetId())

	return next.Server(crossCTX).Close(crossCTX, conn)
}
