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

// Package crossnse provides a NetworkServiceServer chain element that tracks local Cross connect Endpoints and call them first
// their unix file socket as the clienturl.ClientURL(ctx) used to connect to them.
package crossnse

import (
	"context"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/registry/common/crossnse"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type crossConnectNSEServer struct {
	// Map of names -> *registry.NetworkServiceEndpoint for local bypass to file crossNSEs
	crossNSEs sync.Map

	activeConnection sync.Map

	nsmgrName string
	nsmgrInit sync.Once
}

type connectionInfo struct {
	endpointURL *url.URL
	crossNSEURL *url.URL
	closingNSE  bool
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
	rv := &crossConnectNSEServer{}
	*registryServer = crossnse.NewNetworkServiceRegistryServer(rv)
	return rv
}

func (l *crossConnectNSEServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (result *networkservice.Connection, err error) {
	// Check if there is no active connection, we need to replace endpoint url with forwarder url
	conn := request.GetConnection()
	ind := conn.GetPath().GetIndex() // It is designed to be used inside Endpoint, so current index is Endpoint already
	connID := conn.GetId()

	if len(conn.GetPath().GetPathSegments()) == 0 || ind <= 0 {
		return nil, errors.Errorf("path segment doesn't have a client or cross connect nse identity")
	}
	// Remember current segment Name if not yet.
	l.nsmgrInit.Do(func() {
		l.nsmgrName = conn.GetPath().GetPathSegments()[ind].Name
	})

	// We need to find an Id from path to match active connection request.
	clientConnID := l.getConnectionID(conn)

	connInfoRaw, ok := l.activeConnection.Load(clientConnID)
	if !ok {
		if connID != clientConnID {
			return nil, errors.Errorf("connection id should match current path segment id")
		}
		// We came from client, so select cross nse and go to it.
		clientURL := clienturl.ClientURL(ctx)

		// Iterate over all cross connect NSEs to check one with passed state.

		l.crossNSEs.Range(func(key, value interface{}) bool {
			crossNSE, ok := value.(*registry.NetworkServiceEndpoint)
			if ok {
				crossNSEURL, _ := url.Parse(crossNSE.Url)
				crossCTX := clienturl.WithClientURL(ctx, crossNSEURL)

				// Store client connection and selected cross connection URL.
				_, _ = l.activeConnection.LoadOrStore(conn.Id, &connectionInfo{
					endpointURL: clientURL,
					crossNSEURL: crossNSEURL,
				})
				result, err = next.Server(crossCTX).Request(crossCTX, request)
				if err != nil {
					trace.Log(ctx).Errorf("failed to request cross NSE %v err: %v", crossNSEURL, err)
				} else {
					// If all is ok, stop iterating.
					return false
				}
			}
			return true
		})
		if result != nil {
			return result, nil
		}
		return nil, errors.Errorf("all cross NSE failed to connect to endpoint %v connection: %v", clientURL, conn)
	}

	// Go to endpoint URL
	connInfo := connInfoRaw.(*connectionInfo)
	crossCTX := clienturl.WithClientURL(ctx, connInfo.endpointURL)
	return next.Server(crossCTX).Request(crossCTX, request)
}

func (l *crossConnectNSEServer) getConnectionID(conn *networkservice.Connection) string {
	id := ""
	for i := conn.GetPath().GetIndex(); i > 0; i-- {
		if conn.GetPath().GetPathSegments()[i].Name == l.nsmgrName {
			id = conn.GetPath().GetPathSegments()[i].Id
		}
	}
	return id
}

func (l *crossConnectNSEServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// We need to find an Id from path to match active connection request.
	id := l.getConnectionID(conn)

	// We came from cross nse, we need to go to proper endpoint
	connInfoRaw, ok := l.activeConnection.Load(id)
	if !ok {
		return nil, errors.Errorf("no active connection found but we called from cross NSE %v", conn)
	}
	connInfo := connInfoRaw.(*connectionInfo)
	if !connInfo.closingNSE {
		// If not closing NSE go to cross connect
		connInfo.closingNSE = true
		crossCTX := clienturl.WithClientURL(ctx, connInfo.crossNSEURL)
		return next.Server(crossCTX).Close(crossCTX, conn)
	}
	// We are closing NSE, go to endpoint here.
	crossCTX := clienturl.WithClientURL(ctx, connInfo.endpointURL)
	return next.Server(crossCTX).Close(crossCTX, conn)
}

func (l *crossConnectNSEServer) LoadOrStore(name string, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, bool) {
	r, ok := l.crossNSEs.LoadOrStore(name, endpoint)
	if ok {
		return r.(*registry.NetworkServiceEndpoint), true
	}
	return nil, false
}

func (l *crossConnectNSEServer) Delete(name string) {
	l.crossNSEs.Delete(name)
}
