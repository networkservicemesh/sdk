// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package dnscontext provides a dns context specific chain element. It just adds dns configs into dns context of connection context.
// It also provides a possibility to use custom dns configs getters for setup users endpoints.
package dnscontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
)

// GetDNSConfigsFunc gets dns configs
type GetDNSConfigsFunc func() []*networkservice.DNSConfig

type dnsContextServer struct {
	configs []*networkservice.DNSConfig
}

// NewServer creates dns context chain server element.
func NewServer(configs ...*networkservice.DNSConfig) networkservice.NetworkServiceServer {
	return &dnsContextServer{configs: configs}
}

func (d *dnsContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.GetConnection().GetContext().DnsContext = new(networkservice.DNSContext)
	}

	for _, config := range d.configs {
		if !dnsutils.ContainsDNSConfig(request.GetConnection().GetContext().GetDnsContext().Configs, config) {
			request.GetConnection().GetContext().GetDnsContext().Configs = append(request.GetConnection().GetContext().GetDnsContext().Configs, config)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (d *dnsContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
