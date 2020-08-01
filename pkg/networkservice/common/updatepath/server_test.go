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

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestNewServer_SetNewConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t)
	server := updatepath.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(connectionID, 1))
	require.NotNil(t, conn)
	require.NoError(t, err)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_SetNewPathId(t *testing.T) {
	defer goleak.VerifyNone(t)
	server := updatepath.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(connectionID, 0))
	require.NotNil(t, conn)
	require.NoError(t, err)
	require.Equal(t, conn.Path.PathSegments[1].Name, "nse-3") // Check name is replaced.
	require.Equal(t, conn.Id, conn.Path.PathSegments[1].Id)
}

func TestNewServer_PathSegmentNameEqualClientName(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := updatepath.NewServer("nse-2")
	conn, err := server.Request(context.Background(), request(connectionID, 1))
	require.NotNil(t, conn)
	require.NoError(t, err)
	require.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_PathSegmentIdEqualConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := updatepath.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	require.NotNil(t, conn)

	require.NoError(t, err)
	require.NotEqual(t, conn.Id, pathSegmentID2)
}

func TestNewServer_PathSegmentNameIDEqualClientNameID(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := updatepath.NewServer("nse-2")
	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	assert.NotNil(t, conn)

	require.NoError(t, err)
	require.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewServer_InvalidIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := updatepath.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(connectionID, 2))
	require.Nil(t, conn)
	require.NotNil(t, err)
}
