// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package failure provides chain elements failing request at specified times
package failure

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type failureServer struct {
	count        int
	failureTimes []int
}

// NewServer returns a new server chain element failing request:
// * [0, 2, 3] - will fail 0, 2, 3 requests
// * [-1] - will fail all requests
// * [1, 4, -1] = will fail 0 request and all requests starting from 4
func NewServer(failureTimes ...int) networkservice.NetworkServiceServer {
	return &failureServer{
		failureTimes: failureTimes,
	}
}

func (s *failureServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	defer func() { s.count++ }()
	for _, failureTime := range s.failureTimes {
		if failureTime > s.count {
			break
		}
		if failureTime == s.count || failureTime == -1 {
			return nil, errors.Errorf("failure #%d", s.count)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *failureServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
