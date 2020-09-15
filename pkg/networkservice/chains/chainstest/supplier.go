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

package chainstest

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Supplier provides API for supplying NSC, NSE
type Supplier struct {
	resources    []context.CancelFunc
	require      *require.Assertions
	supplyNSE    SupplyEndpointFunc
	supplyNSC    SupplyClientFunc
	tokenGenFunc token.GeneratorFunc
	ctx          context.Context
	dialContext  func(ctx context.Context, u *url.URL) *grpc.ClientConn
}

// Cleanup cleanups all used resources
func (s *Supplier) Cleanup() {
	for _, f := range s.resources {
		f()
	}
	s.resources = nil
}

// SupplyNSC creates new networkservice.NetworkServiceClient that connects to passed url
func (s *Supplier) SupplyNSC(name string, connectTo *url.URL) networkservice.NetworkServiceClient {
	return s.supplyNSC(s.ctx, name, nil, s.tokenGenFunc, s.dialContext(s.ctx, connectTo))
}

// SupplyNSE creates new NSE for passed network service manager
func (s *Supplier) SupplyNSE(registration *registryapi.NetworkServiceEndpoint, mgr nsmgr.Nsmgr) *EndpointEntry {
	nse := s.supplyNSE(s.ctx, registration, s.tokenGenFunc)
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(s.ctx, serveURL, nse.Register)
	_, err := mgr.NetworkServiceEndpointRegistryServer().Register(context.Background(), &registryapi.NetworkServiceEndpoint{
		Name:                registration.Name,
		Url:                 serveURL.String(),
		NetworkServiceNames: registration.NetworkServiceNames,
	})
	s.require.NoError(err)
	for _, service := range registration.NetworkServiceNames {
		_, err = mgr.NetworkServiceRegistryServer().Register(s.ctx, &registryapi.NetworkService{
			Name:    service,
			Payload: "IP",
		})
		s.require.NoError(err)
	}
	log.Entry(s.ctx).Infof("Endpoint %v listen on: %v", registration.Name, serveURL)
	return &EndpointEntry{Endpoint: nse, URL: serveURL}
}
