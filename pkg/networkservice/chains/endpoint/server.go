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
	"github.com/networkservicemesh/api/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setid"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Endpoint - aggregates the APIs:
//            - networkservice.NetworkServiceServer
//            -networkservice.MonitorConnectionServer
type Endpoint interface {
	networkservice.NetworkServiceServer
	networkservice.MonitorConnectionServer
	// Register - register the endpoint with *grpc.Server s
	Register(s *grpc.Server)
}

type endpoint struct {
	networkservice.NetworkServiceServer
	networkservice.MonitorConnectionServer
}

// NewServer - returns a NetworkServiceMesh client as a chain of the standard Client pieces plus whatever
//             additional functionality is specified
//             - name - name of the NetworkServiceServer
//             - authzServer authorization server chain element
//             - tokenGenerator - token.GeneratorFunc - generates tokens for use in Path
//             - additionalFunctionality - any additional NetworkServiceServer chain elements to be included in the chain
func NewServer(name string, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, additionalFunctionality ...networkservice.NetworkServiceServer) Endpoint {
	rv := &endpoint{}
	var ns networkservice.NetworkServiceServer = rv
	rv.NetworkServiceServer = chain.NewNetworkServiceServer(
		append([]networkservice.NetworkServiceServer{
			authzServer,
			setid.NewServer(name),
			monitor.NewServer(&rv.MonitorConnectionServer),
			timeout.NewServer(&ns),
			updatepath.NewServer(name, tokenGenerator),
		}, additionalFunctionality...)...)
	return rv
}

func (e *endpoint) Register(s *grpc.Server) {
	networkservice.RegisterNetworkServiceServer(s, e)
	networkservice.RegisterMonitorConnectionServer(s, e)
	// Create GRPC Health Server:
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	for _, service := range api.ServiceNames(e) {
		healthServer.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
	}
}
