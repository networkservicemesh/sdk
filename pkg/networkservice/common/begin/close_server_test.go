// Copyright (c) 2022 Cisco and/or its affiliates.
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

package begin_test

import (
	"context"
	"sync"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestCloseServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		&markServer{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := server.Request(ctx, testRequest(id))
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	assert.Equal(t, conn.GetContext().GetExtraContext()[mark], mark)
	conn = conn.Clone()
	delete(conn.GetContext().GetExtraContext(), mark)
	assert.Zero(t, conn.GetContext().GetExtraContext()[mark])
	_, err = server.Close(ctx, conn)
	assert.NoError(t, err)
}

type markServer struct {
	t *testing.T
}

func (m *markServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	if request.GetConnection().GetContext().GetExtraContext() == nil {
		request.GetConnection().GetContext().ExtraContext = make(map[string]string)
	}
	request.GetConnection().GetContext().GetExtraContext()[mark] = mark
	return next.Server(ctx).Request(ctx, request)
}

func (m *markServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	assert.NotNil(m.t, conn.GetContext().GetExtraContext())
	assert.Equal(m.t, mark, conn.GetContext().GetExtraContext()[mark])
	return next.Server(ctx).Close(ctx, conn)
}

var _ networkservice.NetworkServiceServer = &markServer{}

func TestDoubleCloseServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := chain.NewNetworkServiceServer(
		begin.NewServer(),
		&doubleCloseServer{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := server.Request(ctx, testRequest(id))
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	conn = conn.Clone()
	_, err = server.Close(ctx, conn)
	assert.NoError(t, err)
	_, err = server.Close(ctx, conn)
	assert.NoError(t, err)
}

type doubleCloseServer struct {
	t *testing.T
	sync.Once
}

func (s *doubleCloseServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return next.Server(ctx).Request(ctx, request)
}

func (s *doubleCloseServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	count := 1
	s.Do(func() {
		count++
	})
	assert.Equal(s.t, 2, count, "Close has been called more than once")
	return next.Server(ctx).Close(ctx, conn)
}

var _ networkservice.NetworkServiceClient = &doubleCloseClient{}
