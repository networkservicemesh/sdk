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

package switchcase

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// ServerCase is a case type for the switch-case server chain element.
type ServerCase struct {
	Condition Condition
	Server    networkservice.NetworkServiceServer
}

type switchServer struct {
	cases []*ServerCase
}

// NewServer returns a new switch-case server chain element.
func NewServer(cases ...*ServerCase) networkservice.NetworkServiceServer {
	return &switchServer{
		cases: cases,
	}
}

func (s *switchServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	for _, c := range s.cases {
		if c.Condition(ctx, request.GetConnection()) {
			return c.Server.Request(ctx, request)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *switchServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	for _, c := range s.cases {
		if c.Condition(ctx, conn) {
			return c.Server.Close(ctx, conn)
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
