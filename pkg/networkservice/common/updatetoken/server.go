// Copyright (c) 2020-2022 Cisco Systems, Inc.
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

// Package updatetoken provides chain elements to update Connection.Path
package updatetoken

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type updateTokenServer struct {
	tokenGenerator token.GeneratorFunc
}

// NewServer - creates a NetworkServiceServer chain element to update the Connection token information
//   - name - the name of the NetworkServiceServer of which the chain element is part
func NewServer(tokenGenerator token.GeneratorFunc) networkservice.NetworkServiceServer {
	return &updateTokenServer{
		tokenGenerator: tokenGenerator,
	}
}

func (u *updateTokenServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if prev := request.GetConnection().GetPrevPathSegment(); prev != nil {
		var tok, expireTime, err = token.FromContext(ctx)

		if err != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", err)
		} else {
			expires := timestamppb.New(expireTime.Local())

			prev.Expires = expires
			prev.Token = tok
		}
	}
	if request.Connection == nil {
		request.Connection = &networkservice.Connection{}
	}
	err := updateToken(ctx, request.GetConnection(), u.tokenGenerator)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (u *updateTokenServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if prev := conn.GetPrevPathSegment(); prev != nil {
		var tok, expireTime, err = token.FromContext(ctx)

		if err != nil {
			log.FromContext(ctx).Warnf("an error during getting token from the context: %+v", err)
		} else {
			expires := timestamppb.New(expireTime.Local())

			prev.Expires = expires
			prev.Token = tok
		}
	}
	err := updateToken(ctx, conn, u.tokenGenerator)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
