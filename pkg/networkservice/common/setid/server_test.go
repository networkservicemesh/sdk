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

package setid_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setid"
)

func TestNewServer_SetNewConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := setid.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(connectionID, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_PathSegmentNameEqualClientName(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := setid.NewServer("nse-2")
	conn, err := server.Request(context.Background(), request(connectionID, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}

func TestNewServer_PathSegmentIdEqualConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := setid.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, pathSegmentID2)
}

func TestNewServer_PathSegmentNameIDEqualClientNameID(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := setid.NewServer("nse-2")
	conn, err := server.Request(context.Background(), request(pathSegmentID2, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewServer_InvalidIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := setid.NewServer("nse-3")
	conn, err := server.Request(context.Background(), request(connectionID, 2))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, connectionID)
}
