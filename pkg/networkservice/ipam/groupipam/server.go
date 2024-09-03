// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package groupipam provides a networkservice.NetworkServiceServer chain element to handle a group of []*net.IPNet.
// The chain element should be used when the endpoint should assign a few addresses for the connection.
// By default `groupipam` uses `point2pointipam` to handle *net.IPNet.
package groupipam

import (
	"net"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
)

type options struct {
	newIPAMServerFn func(...*net.IPNet) networkservice.NetworkServiceServer
}

// Option allows to change a default behavior.
type Option func(*options)

// WithCustomIPAMServer replaces default `point2pointipam` to custom implementation.
func WithCustomIPAMServer(f func(...*net.IPNet) networkservice.NetworkServiceServer) Option {
	if f == nil {
		panic("nil is not allowed")
	}

	return func(o *options) {
		o.newIPAMServerFn = f
	}
}

// NewServer creates a new instance of groupipam chain element that handles a group of []*net.IPNet.
// Requires a group of []*net.IPNet.
// Options can be passed optionally.
func NewServer(groups [][]*net.IPNet, opts ...Option) networkservice.NetworkServiceServer {
	var ipamServers []networkservice.NetworkServiceServer
	o := options{
		newIPAMServerFn: point2pointipam.NewServer,
	}

	for _, opt := range opts {
		opt(&o)
	}

	for _, group := range groups {
		ipamServers = append(ipamServers, o.newIPAMServerFn(group...))
	}

	return chain.NewNetworkServiceServer(ipamServers...)
}
