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

package authorize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type authorizeClient struct {
	requestPolicy func(peer *peer.Peer, conn *networkservice.Connection) error
}

// NewClient - returns a new authorization networkservicemesh.NetworkServiceClient
//             - requestPolicy - function that takes the peer and Connection returned from the server and returns a non-nil error if unauthorized
func NewClient(requestPolicy func(peer *peer.Peer, conn *networkservice.Connection) error) networkservice.NetworkServiceClient {
	return &authorizeClient{requestPolicy: requestPolicy}
}

func (a *authorizeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if a.requestPolicy == nil {
		return nil, errors.WithStack(status.Error(codes.Unavailable, "requestPolicy is nil, request cannot authorize any connection returned from server"))
	}
	p, _ := peer.FromContext(ctx)
	err = a.requestPolicy(p, conn)
	if err != nil {
		return nil, errors.WithStack(status.Errorf(codes.PermissionDenied, "permission denied: %+v", err))
	}
	return conn, err
}

func (a *authorizeClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Note: There's no point in authorizing the return to a Close...
	return next.Client(ctx).Close(ctx, conn, opts...)
}
