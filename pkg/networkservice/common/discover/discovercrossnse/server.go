// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package discovercrossnse

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type switchNetworkServiceEndpointNameServer struct {
	setFunc func(context.Context, string)
	getFunc func(context.Context) string
}

func (d *switchNetworkServiceEndpointNameServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	d.setFunc(ctx, request.GetConnection().GetNetworkServiceEndpointName())

	request.GetConnection().NetworkServiceEndpointName = d.getFunc(ctx)

	return next.Server(ctx).Request(ctx, request)
}

func (d *switchNetworkServiceEndpointNameServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	d.setFunc(ctx, conn.NetworkServiceEndpointName)

	conn.NetworkServiceEndpointName = d.getFunc(ctx)

	return next.Server(ctx).Close(ctx, conn)
}

// NewServer TODO
func NewServer(nsClient registry.NetworkServiceRegistryClient, nseClient registry.NetworkServiceEndpointRegistryClient, loadBalance networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		&switchNetworkServiceEndpointNameServer{
			setFunc: storeNetworkServiceEndpointName,
			getFunc: loadCrossConnectEndpointName,
		},
		discover.NewServer(nsClient, nseClient),
		loadBalance,
		&switchNetworkServiceEndpointNameServer{
			setFunc: storeCrossConnectEndpointName,
			getFunc: loadNetworkServiceEndpointName,
		},
	)
}
