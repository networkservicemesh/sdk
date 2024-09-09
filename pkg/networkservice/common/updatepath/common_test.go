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

package updatepath_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

const (
	nse1           = "nse-1"
	nse2           = "nse-2"
	nse3           = "nse-3"
	pathSegmentID1 = "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9"
	pathSegmentID2 = "ece490ea-dfe8-4512-a3ca-5be7b39515c5"
	pathSegmentID3 = "f9a83e55-0a4f-3647-144a-98a9ee8fb231"
	connectionID   = "54ec76a7-d642-43ca-87bc-ab51765f575a"
)

func request(connectionID string, path *networkservice.Path) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:   connectionID,
			Path: path,
		},
	}
}

func path(pathIndex uint32, pathSegments int) *networkservice.Path {
	if pathSegments == 0 {
		return nil
	}

	path := &networkservice.Path{
		Index: pathIndex,
	}
	if pathSegments >= 1 {
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{
			Name: nse1,
			Id:   pathSegmentID1,
		})
	}
	if pathSegments >= 2 {
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{
			Name: nse2,
			Id:   pathSegmentID2,
		})
	}
	if pathSegments >= 3 {
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{
			Name: nse3,
			Id:   pathSegmentID3,
		})
	}
	return path
}

func requirePathEqual(t *testing.T, expected, actual *networkservice.Path, unknownIDs ...int) {
	expected = expected.Clone()
	actual = actual.Clone()
	for _, index := range unknownIDs {
		expected.PathSegments[index].Id = ""
		actual.PathSegments[index].Id = ""
	}
	require.Equal(t, expected.String(), actual.String())
}

type sample struct {
	name string
	test func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer)
}

var samples = []*sample{
	{
		name: "NoPathNoID",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse1)

			conn, err := server.Request(context.Background(), request("", nil))
			require.NoError(t, err)
			require.NotNil(t, conn)

			requirePathEqual(t, path(0, 1), conn.GetPath(), 0)

			require.Equal(t, conn.GetId(), conn.GetPath().GetPathSegments()[0].GetId())
		},
	},
	{
		name: "NoPath",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse1)

			conn, err := server.Request(context.Background(), request(connectionID, nil))
			require.NoError(t, err)
			require.NotNil(t, conn)

			path := path(0, 1)
			path.PathSegments[0].Id = connectionID
			requirePathEqual(t, path, conn.GetPath())

			require.Equal(t, connectionID, conn.GetId())
		},
	},
	{
		name: "SameNameDifferentID",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse2)

			conn, err := server.Request(context.Background(), request(connectionID, path(1, 2)))
			require.NoError(t, err)
			require.NotNil(t, conn)

			requirePathEqual(t, path(1, 2), conn.GetPath())

			require.Equal(t, pathSegmentID2, conn.GetId())
		},
	},
	{
		name: "DifferentNameInvalidID",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			_, err := server.Request(context.Background(), request(connectionID, path(1, 2)))
			require.Error(t, err)
		},
	},
	{
		name: "InvalidIndex",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := newUpdatePathServer(nse3)

			_, err := server.Request(context.Background(), request(pathSegmentID2, path(3, 2)))
			require.Error(t, err)
		},
	},
	{
		name: "DifferentNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var connPath *networkservice.Path
			server := next.NewNetworkServiceServer(
				newUpdatePathServer(nse3),
				checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
					connPath = request.GetConnection().GetPath()
					requirePathEqual(t, path(2, 3), connPath, 2)
					require.Equal(t, connPath.GetPathSegments()[2].GetId(), request.GetConnection().GetId())
					require.NotEqual(t, pathSegmentID3, request.GetConnection().GetId())
					require.NotEqual(t, pathSegmentID2, request.GetConnection().GetId())
				}),
			)

			requestPath := path(1, 3)
			requestPath.PathSegments[2].Name = "different"
			conn, err := server.Request(context.Background(), request(pathSegmentID2, requestPath))
			require.NoError(t, err)
			require.NotNil(t, conn)

			connPath.Index = 1
			requirePathEqual(t, connPath, conn.GetPath(), 2)

			require.Equal(t, pathSegmentID2, conn.GetId())
		},
	},
	{
		name: "NoNextAvailable",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			var connPath *networkservice.Path
			server := next.NewNetworkServiceServer(
				newUpdatePathServer(nse3),
				checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
					connPath = request.GetConnection().GetPath()
					requirePathEqual(t, path(2, 3), connPath, 2)
					require.Equal(t, connPath.GetPathSegments()[2].GetId(), request.GetConnection().GetId())
					require.NotEqual(t, pathSegmentID2, request.GetConnection().GetId())
				}),
			)

			conn, err := server.Request(context.Background(), request(pathSegmentID2, path(1, 2)))
			require.NoError(t, err)
			require.NotNil(t, conn)

			connPath.Index = 1
			requirePathEqual(t, connPath, conn.GetPath(), 2)

			require.Equal(t, pathSegmentID2, conn.GetId())
		},
	},
	{
		name: "SameNextName",
		test: func(t *testing.T, newUpdatePathServer func(name string) networkservice.NetworkServiceServer) {
			t.Cleanup(func() {
				goleak.VerifyNone(t)
			})

			server := next.NewNetworkServiceServer(
				newUpdatePathServer(nse3),
				checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
					requirePathEqual(t, path(2, 3), request.GetConnection().GetPath())
					require.Equal(t, pathSegmentID3, request.GetConnection().GetId())
				}),
			)

			conn, err := server.Request(context.Background(), request(pathSegmentID2, path(1, 3)))
			require.NoError(t, err)
			require.NotNil(t, conn)

			requirePathEqual(t, path(1, 3), conn.GetPath())

			require.Equal(t, pathSegmentID2, conn.GetId())
		},
	},
}

func TestUpdatePath(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run("TestNewServer_"+sample.name, func(t *testing.T) {
			sample.test(t, updatepath.NewServer)
		})
	}
	for i := range samples {
		sample := samples[i]
		t.Run("TestNewClient_"+sample.name, func(t *testing.T) {
			sample.test(t, func(name string) networkservice.NetworkServiceServer {
				return adapters.NewClientToServer(updatepath.NewClient(name))
			})
		})
	}
}
