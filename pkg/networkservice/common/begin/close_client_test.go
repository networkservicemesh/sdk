// Copyright (c) 2021 Cisco and/or its affiliates.
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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	mark = "mark"
)

func TestCloseClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		&markClient{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := client.Request(ctx, testRequest(id))
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	assert.Equal(t, conn.GetContext().GetExtraContext()[mark], mark)
	conn = conn.Clone()
	delete(conn.GetContext().GetExtraContext(), mark)
	assert.Zero(t, conn.GetContext().GetExtraContext()[mark])
	_, err = client.Close(ctx, conn)
	assert.NoError(t, err)
}

type markClient struct {
	t *testing.T
}

func (m *markClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	if request.GetConnection().GetContext().GetExtraContext() == nil {
		request.GetConnection().GetContext().ExtraContext = make(map[string]string)
	}
	request.GetConnection().GetContext().GetExtraContext()[mark] = mark
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (m *markClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	assert.NotNil(m.t, conn.GetContext().GetExtraContext())
	assert.Equal(m.t, mark, conn.GetContext().GetExtraContext()[mark])
	return next.Client(ctx).Close(ctx, conn, opts...)
}

var _ networkservice.NetworkServiceClient = &markClient{}

func TestDoubleCloseClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		&doubleCloseClient{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := client.Request(ctx, testRequest(id))
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	conn = conn.Clone()
	_, err = client.Close(ctx, conn)
	assert.NoError(t, err)
	_, err = client.Close(ctx, conn)
	assert.NoError(t, err)
}

type doubleCloseClient struct {
	t *testing.T
	sync.Once
}

func (s *doubleCloseClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *doubleCloseClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	count := 1
	s.Do(func() {
		count++
	})
	assert.Equal(s.t, 2, count, "Close has been called more than once")
	return next.Client(ctx).Close(ctx, conn, opts...)
}

var _ networkservice.NetworkServiceClient = &doubleCloseClient{}
