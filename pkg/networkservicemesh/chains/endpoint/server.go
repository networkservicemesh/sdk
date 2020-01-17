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

// Package endpoint provides a simple wrapper for building a NetworkServiceServer
package endpoint

import (
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/common/setid"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/chain"
)

// Endpoint - aggregates the APIs:
//            - networkservice.NetworkServiceServer
//            -connection.MonitorConnectionServer
type Endpoint interface {
	networkservice.NetworkServiceServer
	connection.MonitorConnectionServer
	// Register - register the endpoint with *grpc.Server s
	Register(s *grpc.Server)
}

type endpoint struct {
	networkservice.NetworkServiceServer
	connection.MonitorConnectionServer
}

// NewServer - returns a NetworkServiceMesh client as a chain of the standard Client pieces plus whatever
//             additional functionality is specified
//             - name - name of the NetworkServiceServer
//             - additionalFunctionality - any additional NetworkServiceServer chain elements to be included in the chain
func NewServer(name string, additionalFunctionality ...networkservice.NetworkServiceServer) Endpoint {
	rv := &endpoint{}
	var ns networkservice.NetworkServiceServer = rv
	rv.NetworkServiceServer = chain.NewNetworkServiceServer(
		append([]networkservice.NetworkServiceServer{
			authorize.NewServer(),
			setid.NewServer(name),
			monitor.NewServer(&rv.MonitorConnectionServer),
			timeout.NewServer(&ns),
			updatepath.NewServer(name),
		}, additionalFunctionality...)...)
	return rv
}

func (e *endpoint) Register(s *grpc.Server) {
	networkservice.RegisterNetworkServiceServer(s, e)
	connection.RegisterMonitorConnectionServer(s, e)
}
