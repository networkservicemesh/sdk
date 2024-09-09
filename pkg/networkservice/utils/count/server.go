// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// Server is a server type for counting Requests/Closes.
type Server struct {
	totalRequests, TotalCloses int32
	requests, closes           map[string]int32
	mu                         sync.Mutex
}

// Request performs request and increments requests count.
func (s *Server) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.mu.Lock()

	atomic.AddInt32(&s.totalRequests, 1)
	if s.requests == nil {
		s.requests = make(map[string]int32)
	}
	s.requests[request.GetConnection().GetId()]++
	s.mu.Unlock()

	return next.Server(ctx).Request(ctx, request)
}

// Close performs close and increments closes count.
func (s *Server) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	s.mu.Lock()

	atomic.AddInt32(&s.TotalCloses, 1)
	if s.closes == nil {
		s.closes = make(map[string]int32)
	}
	s.closes[connection.GetId()]++
	s.mu.Unlock()

	return next.Server(ctx).Close(ctx, connection)
}

// Requests returns requests count.
func (s *Server) Requests() int {
	return int(atomic.LoadInt32(&s.totalRequests))
}

// Closes returns closes count.
func (s *Server) Closes() int {
	return int(atomic.LoadInt32(&s.TotalCloses))
}

// UniqueRequests returns unique requests count.
func (s *Server) UniqueRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.requests == nil {
		return 0
	}
	return len(s.requests)
}

// UniqueCloses returns unique closes count.
func (s *Server) UniqueCloses() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closes == nil {
		return 0
	}
	return len(s.closes)
}
