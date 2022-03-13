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

// mergeConnection - deals explicitly with the problem of "What should we allow the 'main' of a client executable
// to update from outside the chain after the initial request for a connection.
// One minor point to keep in mind here is that a passthrough NSE, one that is a server that incorporates a client
// into its own chain, begin's at the server side... and so the server's use of client like functions are 'inside the chain'
func mergeConnection(returnedConnection, requestedConnection *networkservice.Connection) *networkservice.Connection {
	// If this request has has yet to return successfully, go with the requesteConnection
	if returnedConnection == nil {
		return requestedConnection
	}

	// Clone the previously returned connection
	conn := returnedConnection.Clone()

	// If the Request is asking for a new NSE, use that
	if returnedConnection.GetNetworkServiceEndpointName() != requestedConnection.GetNetworkServiceEndpointName() {
		conn.NetworkServiceEndpointName = requestedConnection.GetNetworkServiceEndpointName()
	}

	// Ifthe Request is asking for a change in ConnectionContext, propagate that.
	if !proto.Equal(returnedConnection.GetContext(), requestedConnection.GetContext()) {
		conn.Context = proto.Clone(requestedConnection.GetContext()).(*networkservice.ConnectionContext)
	}

	// Note: We are disallowing at this time changes in requested NetworkService, Mechanism, Labels, or Path.
	// In the future it may be worth permitting changes in the Labels.
	// It probably is not a good idea to allow changes in the NetworkService or Path here.

	return conn
}
