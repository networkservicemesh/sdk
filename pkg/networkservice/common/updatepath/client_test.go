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

// Package updatepath_test provides a tests for package 'setid'
package updatepath_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

const (
	pathSegmentID1 string = "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9"
	pathSegmentID2 string = "ece490ea-dfe8-4512-a3ca-5be7b39515c5"
	connectionID   string = "54ec76a7-d642-43ca-87bc-ab51765f575a"
)

func request(connectionID string, pathIndex uint32) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connectionID,
			Path: &networkservice.Path{
				Index: pathIndex,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nse-1",
						Id:   pathSegmentID1,
					},
					{
						Name: "nse-2",
						Id:   pathSegmentID2,
					},
				},
			},
		},
	}
}

func TestNewClient_SetNewConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	client := updatepath.NewClient("nse-3")

	conn, err := client.Request(context.Background(), request(connectionID, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 3, len(conn.Path.PathSegments))
	require.Equal(t, "nse-3", conn.Path.PathSegments[2].Name)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_SetNewConnectionId2(t *testing.T) {
	defer goleak.VerifyNone(t)

	client := next.NewNetworkServiceClient(
		updatepath.NewClient("nse-2"),
		checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			require.Equal(t, 1, int(request.Connection.Path.Index))
		}),
	)

	conn, err := client.Request(context.Background(), request(connectionID, 0))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewClient_SetNewConnectionIdSecondReq(t *testing.T) {
	defer goleak.VerifyNone(t)

	client := updatepath.NewClient("nse-1")

	conn, err := client.Request(context.Background(), request(connectionID, 0))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Id, pathSegmentID1)

	firstConnID := conn.Id

	conn, err = client.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: conn,
	})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, firstConnID, conn.Id)
}

func TestNewClient_PathSegmentNameEqualClientName(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	client := updatepath.NewClient("nse-2")

	conn, err := client.Request(context.Background(), request(connectionID, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_PathSegmentIdEqualConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	client := updatepath.NewClient("nse-2")

	conn, err := client.Request(context.Background(), request(pathSegmentID2, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewClient_PathSegmentNameAndIDEqualClientNameAndID(t *testing.T) {
	defer goleak.VerifyNone(t)

	client := updatepath.NewClient("nse-2")

	conn, err := client.Request(context.Background(), request(pathSegmentID2, 1))
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewClient_InvalidIndex(t *testing.T) {
	defer goleak.VerifyNone(t)

	client := updatepath.NewClient("nse-3")

	conn, err := client.Request(context.Background(), request(connectionID, 2))
	require.Error(t, err)
	require.Nil(t, conn)
}

func TestNewClient_NoConnection(t *testing.T) {
	defer goleak.VerifyNone(t)

	client := updatepath.NewClient("nse-3")

	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_RestoreIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	client := next.NewNetworkServiceClient(
		updatepath.NewClient("nse-1"),
		newPathIndexClient(t, 0),
		updatepath.NewClient("nse-2"),
		newPathIndexClient(t, 1),
		updatepath.NewClient("nse-2"),
		newPathIndexClient(t, 1),
		updatepath.NewClient("nse-3"),
		newPathIndexClient(t, 2),
	)

	_, err := client.Request(context.Background(), request(connectionID, 0))
	require.NoError(t, err)
}

type pathIndexClient struct {
	t     *testing.T
	index uint32
}

func newPathIndexClient(t *testing.T, index uint32) *pathIndexClient {
	return &pathIndexClient{
		t:     t,
		index: index,
	}
}

func (c *pathIndexClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	require.Equal(c.t, c.index, request.Connection.Path.Index)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	require.Equal(c.t, c.index, conn.Path.Index)
	return conn, nil
}

func (c *pathIndexClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
