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

// Package macaddr provides chain element to set EthernetContext.SrcMacAddr
package macaddr

import (
	"context"
	"net"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Supplier is a type for net.HardwareAddr supplier
type Supplier func() net.HardwareAddr

type macAddrServer struct {
	supplier Supplier
}

// NewServer returns a new MAC address server chain element
func NewServer(supplier Supplier) networkservice.NetworkServiceServer {
	if supplier == nil {
		supplier = randMac
	}

	return &macAddrServer{
		supplier: supplier,
	}
}

func (s *macAddrServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	if request.GetConnection().GetContext().GetEthernetContext() == nil {
		request.GetConnection().GetContext().EthernetContext = &networkservice.EthernetContext{}
	}
	if request.GetConnection().GetContext().GetEthernetContext().GetSrcMac() == "" {
		request.GetConnection().GetContext().GetEthernetContext().SrcMac = s.supplier().String()
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *macAddrServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
