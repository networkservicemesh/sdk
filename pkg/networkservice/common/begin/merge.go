// Copyright (c) 2021-2022 Cisco and/or its affiliates.
// Copyright (c) 2022 Nordix Foundation
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

package begin

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/protobuf/proto"
)

func mergeConnection(returnedConnection, requestedConnection, connection *networkservice.Connection) *networkservice.Connection {
	if returnedConnection == nil || connection == nil {
		return requestedConnection
	}
	conn := connection.Clone()
	if returnedConnection.GetNetworkServiceEndpointName() != requestedConnection.GetNetworkServiceEndpointName() {
		conn.NetworkServiceEndpointName = requestedConnection.GetNetworkServiceEndpointName()
	}
	conn.Context = mergeConnectionContext(returnedConnection.GetContext(), requestedConnection.GetContext(), connection.GetContext())
	return conn
}

func mergeConnectionContext(returnedConnectionContext, requestedConnectionContext, connectioncontext *networkservice.ConnectionContext) *networkservice.ConnectionContext {
	rv := proto.Clone(connectioncontext).(*networkservice.ConnectionContext)
	if !proto.Equal(returnedConnectionContext, requestedConnectionContext) {
		// TODO: DNSContext, EthernetContext, do we need to do MTU?
		rv.IpContext = requestedConnectionContext.IpContext
		rv.ExtraContext = mergeMapStringString(returnedConnectionContext.GetExtraContext(), requestedConnectionContext.GetExtraContext(), connectioncontext.GetExtraContext())
	}
	return rv
}

func mergeMapStringString(returnedMap, requestedMap, mapMap map[string]string) map[string]string {
	// clone the map
	rv := make(map[string]string)
	for k, v := range mapMap {
		rv[k] = v
	}

	for k, v := range returnedMap {
		requestedValue, ok := requestedMap[k]
		// If a key is in returnedMap and its value differs from requestedMap, update the value
		if ok && requestedValue != v {
			rv[k] = requestedValue
		}
		// If a key is in returnedMap and not in requestedMap, delete it
		if !ok {
			delete(rv, k)
		}
	}

	return rv
}
