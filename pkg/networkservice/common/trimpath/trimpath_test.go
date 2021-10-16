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

package trimpath_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/trimpath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

// updatepath.NewServer() should 'trim' the path to terminate with itself if and only if it's not a 'passthrough'
// server.  A 'passthrough' server is one that continues the Path by having a client that has a single outgoing connection
// corresponding to each incoming event to the server that shares the same PathSegment

func TestDontTrimPath(t *testing.T) {
	name := "dontTrim"
	server := chain.NewNetworkServiceServer(
		updatepath.NewServer(name),
		metadata.NewServer(),
		trimpath.NewServer(),
		adapters.NewClientToServer(
			chain.NewNetworkServiceClient(
				updatepath.NewClient(name),
				trimpath.NewClient(),
			),
		),
	)
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, uint32(0), conn.GetPath().GetIndex())
	require.Len(t, conn.GetPath().GetPathSegments(), 1)

	request := &networkservice.NetworkServiceRequest{}
	request.Connection = conn.Clone()
	request.GetConnection().GetPath().PathSegments = append(request.GetConnection().GetPath().PathSegments, &networkservice.PathSegment{})

	conn2, err := server.Request(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, uint32(0), conn2.GetPath().GetIndex())
	require.Len(t, conn2.GetPath().GetPathSegments(), 2)
}

func TestTrimPath(t *testing.T) {
	name := "trim"
	server := chain.NewNetworkServiceServer(
		updatepath.NewServer(name),
		metadata.NewServer(),
		trimpath.NewServer(),
	)
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, uint32(0), conn.GetPath().GetIndex())
	require.Len(t, conn.GetPath().GetPathSegments(), 1)

	request := &networkservice.NetworkServiceRequest{}
	request.Connection = conn.Clone()
	request.GetConnection().GetPath().PathSegments = append(request.GetConnection().GetPath().PathSegments, &networkservice.PathSegment{})

	conn2, err := server.Request(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	require.Equal(t, uint32(0), conn2.GetPath().GetIndex())
	require.Len(t, conn2.GetPath().GetPathSegments(), 1)
}
