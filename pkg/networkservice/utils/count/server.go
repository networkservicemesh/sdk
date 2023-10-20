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

package count

import (
	"context"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Server is a server type for counting Requests/Closes
type Server struct {
	totalRequests int32
}

// Request performs request and increments requests count
func (s *Server) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	atomic.AddInt32(&s.totalRequests, 1)
	return next.Server(ctx).Request(ctx, request)
}

// Close performs close and increments closes count
func (s *Server) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {

	return next.Server(ctx).Close(ctx, connection)
}

// Requests returns requests count
func (s *Server) Requests() int {
	return int(atomic.LoadInt32(&s.totalRequests))
}
