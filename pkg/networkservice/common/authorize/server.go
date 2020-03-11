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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type authorizeServer struct {
	requestPolicy func(peer *peer.Peer, conn *networkservice.NetworkServiceRequest) error
	closePolicy   func(peer *peer.Peer, conn *networkservice.Connection) error
}

// NewServer - returns a new authorization networkservicemesh.NetworkServiceServers
//             - requestPolicy - function that takes a peer and NetworkServiceRequest and returns a non-nil error if the client is not authorized to make the request
//             - closePolicy - function that takes a peer and a Connection and returns non-nil error if the client is not authorized to close the connection
func NewServer(
	requestPolicy func(peer *peer.Peer, conn *networkservice.NetworkServiceRequest) error,
	closePolicy func(peer *peer.Peer, conn *networkservice.Connection) error) networkservice.NetworkServiceServer {
	return &authorizeServer{
		requestPolicy: requestPolicy,
		closePolicy:   closePolicy,
	}
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if a.requestPolicy == nil {
		return nil, errors.WithStack(status.Error(codes.Unavailable, "requestPolicy is nil, request cannot be authorized"))
	}
	p, _ := peer.FromContext(ctx)
	err := a.requestPolicy(p, request)
	if err != nil {
		return nil, errors.WithStack(status.Errorf(codes.PermissionDenied, "permission denied: %+v", err))
	}
	return next.Server(ctx).Request(ctx, request)
}

func (a *authorizeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if a.closePolicy == nil {
		return nil, errors.WithStack(status.Error(codes.Unavailable, "closePolicy is nil, close cannot be authorized"))
	}
	p, _ := peer.FromContext(ctx)
	err := a.closePolicy(p, conn)
	if err != nil {
		return nil, errors.WithStack(status.Errorf(codes.PermissionDenied, "permission denied: %+v", err))
	}
	return next.Server(ctx).Close(ctx, conn)
}
