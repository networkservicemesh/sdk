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

// Package swapip provides chain element to swapping fields of remote mechanisms such as common.SrcIP and common.DstIP
// from internal to external and vice versa on response.
package swapip

import (
	"context"
	"errors"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/externalips"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type swapIPServer struct{}

func (i *swapIPServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	url := clienturlctx.ClientURL(ctx)
	if url == nil {
		return nil, errors.New("url is required for interdomain use-case")
	}
	dstIP, _, err := net.SplitHostPort(clienturlctx.ClientURL(ctx).Host)
	if err != nil {
		return nil, err
	}
	for _, m := range request.MechanismPreferences {
		if m.Cls == cls.REMOTE {
			if externalIP := externalips.FromInternal(ctx, net.ParseIP(m.Parameters[common.SrcIP])); externalIP != nil {
				m.Parameters[common.SrcIP] = externalIP.String()
			}
		}
	}

	if request.Connection.Mechanism != nil {
		if externalIP := externalips.FromInternal(ctx, net.ParseIP(request.Connection.Mechanism.Parameters[common.SrcIP])); externalIP != nil {
			request.Connection.Mechanism.Parameters[common.SrcIP] = externalIP.String()
		}
	}

	nsName, nseName := request.Connection.NetworkService, request.Connection.NetworkServiceEndpointName
	request.Connection.NetworkServiceEndpointName, request.Connection.NetworkService = interdomain.Target(request.Connection.NetworkServiceEndpointName), interdomain.Target(request.Connection.NetworkService)
	response, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if response.Mechanism != nil {
		response.Mechanism.Parameters[common.DstIP] = dstIP
	}
	response.NetworkService = nsName
	response.NetworkServiceEndpointName = nseName
	return response, err
}

func (i *swapIPServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, connection)
}

// NewServer creates new swap chain element. Expects public IP address of node
func NewServer() networkservice.NetworkServiceServer {
	return &swapIPServer{}
}
