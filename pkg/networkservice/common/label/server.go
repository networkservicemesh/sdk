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

package label

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type labelServer struct {
	label, value string
}

// NewServer returns new server marking Connections with label
func NewServer(label, value string) networkservice.NetworkServiceServer {
	return &labelServer{
		label: label,
		value: value,
	}
}

func (s *labelServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	store(ctx, s.label, s.value)
	return next.Server(ctx).Request(ctx, request)
}

func (s *labelServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	store(ctx, s.label, s.value)
	return next.Server(ctx).Close(ctx, conn)
}
