// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

// mergeConnection - preforms the three way merge of the returnedConnection, requestedConnection and connection
//
//	returnedConnection - the Connection last returned from the begin.Request(...)
//	requestedConnection - the Connection passed in to the begin.Request(...)
//	currentConnection - the last value for the Connection in EventFactory.  Since Refreshes, Heals, etc
//	             can result in changes that have *not* been returned from begin.Request(...) because
//	             they originated in events internal to the chain (instead of external via calls to
//	             begin.Request(...)) it is possible that connection differs from returnedConnection
func mergeConnection(returnedConnection, requestedConnection, currentConnection *networkservice.Connection) *networkservice.Connection {
	if returnedConnection == nil || currentConnection == nil {
		return requestedConnection
	}
	conn := currentConnection.Clone()
	if returnedConnection.GetNetworkServiceEndpointName() != requestedConnection.GetNetworkServiceEndpointName() {
		conn.NetworkServiceEndpointName = requestedConnection.GetNetworkServiceEndpointName()
	}
	conn.Context = mergeConnectionContext(returnedConnection.GetContext(), requestedConnection.GetContext(), currentConnection.GetContext())
	return conn
}

func mergeConnectionContext(returnedConnectionContext, requestedConnectionContext, currentConnectionContext *networkservice.ConnectionContext) *networkservice.ConnectionContext {
	if currentConnectionContext == nil {
		return requestedConnectionContext
	}
	rv := proto.Clone(currentConnectionContext).(*networkservice.ConnectionContext)
	if !proto.Equal(returnedConnectionContext, requestedConnectionContext) {
		rv.IpContext = requestedConnectionContext.GetIpContext()
		rv.EthernetContext = requestedConnectionContext.GetEthernetContext()
		rv.DnsContext = requestedConnectionContext.GetDnsContext()
		rv.MTU = requestedConnectionContext.GetMTU()
		rv.ExtraContext = mergeMapStringString(returnedConnectionContext.GetExtraContext(), requestedConnectionContext.GetExtraContext(), currentConnectionContext.GetExtraContext())
	}
	return rv
}

func mergeMapStringString(returnedMap, requestedMap, currentMap map[string]string) map[string]string {
	// clone the currentMap
	rv := make(map[string]string)
	for k, v := range currentMap {
		rv[k] = v
	}

	// Only intentional changes between the returnedMap (which was values last returned from calls to begin.Request(...))
	// and requestedMap (the values passed into begin.Request for this call) are considered for application to the existing
	// map (currentMap - the last set of values remembered by the EventFactory).
	for k, v := range returnedMap {
		srcValue, ok := requestedMap[k]
		// If a key is in returnedMap and its value differs from requestedMap, update the value
		if ok && srcValue != v {
			rv[k] = srcValue
		}
		// If a key is in returnedMap and not in requestedMap, delete it
		if !ok {
			delete(rv, k)
		}
	}

	return rv
}
