// Copyright (c) 2019-2020 VMware, Inc.
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

package filtermechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/connection"
	"github.com/networkservicemesh/api/pkg/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type filterMechanismsServer struct{}

// NewServer - filters out remote mechanisms if connection is received from a unix file socket, otherwise filters
// out local mechanisms
func NewServer() networkservice.NetworkServiceServer {
	return &filterMechanismsServer{}
}

func (f *filterMechanismsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	p, ok := peer.FromContext(ctx)
	if ok {
		if p.Addr.Network() == "unix" {
			var mechanisms []*connection.Mechanism
			for _, mechanism := range request.GetMechanismPreferences() {
				if mechanism.Cls == cls.LOCAL {
					mechanisms = append(mechanisms, mechanism)
				}
			}
			request.MechanismPreferences = mechanisms
			return next.Server(ctx).Request(ctx, request)
		}
		var mechanisms []*connection.Mechanism
		for _, mechanism := range request.GetMechanismPreferences() {
			if mechanism.Cls == cls.REMOTE {
				mechanisms = append(mechanisms, mechanism)
			}
		}
		request.MechanismPreferences = mechanisms
	}
	return next.Server(ctx).Request(ctx, request)
}

func (f *filterMechanismsServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
