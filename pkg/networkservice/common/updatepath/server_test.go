// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package updatepath_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestNewServer_SetNewConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t)

	server := updatepath.NewServer("nse-3")

	conn, err := server.Request(context.Background(), request(connectionID, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_SetNewPathId(t *testing.T) {
	defer goleak.VerifyNone(t)

	server := updatepath.NewServer("nse-3")

	conn, err := server.Request(context.Background(), request(connectionID, 0))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Path.PathSegments[1].Name, "nse-3") // Check name is replaced.
	require.Equal(t, conn.Id, conn.Path.PathSegments[1].Id)
}

func TestNewServer_PathSegmentNameEqualClientName(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := updatepath.NewServer("nse-2")

	conn, err := server.Request(context.Background(), request(connectionID, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_PathSegmentIdEqualConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := updatepath.NewServer("nse-3")

	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NotEqual(t, conn.Id, pathSegmentID2)
}

func TestNewServer_PathSegmentNameIDEqualClientNameID(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := updatepath.NewServer("nse-2")

	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewServer_InvalidIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := updatepath.NewServer("nse-3")

	conn, err := server.Request(context.Background(), request(connectionID, 2))
	require.Error(t, err)
	require.Nil(t, conn)
}

func TestNewServer_RestoreIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := next.NewNetworkServiceServer(
		updatepath.NewServer("nse-1"),
		newPathIndexServer(t, 0),
		updatepath.NewServer("nse-2"),
		newPathIndexServer(t, 1),
		updatepath.NewServer("nse-2"),
		newPathIndexServer(t, 1),
		updatepath.NewServer("nse-3"),
		newPathIndexServer(t, 2),
	)

	_, err := server.Request(context.Background(), request(connectionID, 0))
	require.NoError(t, err)
}

type pathIndexServer struct {
	t     *testing.T
	index uint32
}

func newPathIndexServer(t *testing.T, index uint32) *pathIndexServer {
	return &pathIndexServer{
		t:     t,
		index: index,
	}
}

func (s *pathIndexServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	require.Equal(s.t, s.index, request.Connection.Path.Index)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	require.Equal(s.t, s.index, conn.Path.Index)
	return conn, nil
}

func (s *pathIndexServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
