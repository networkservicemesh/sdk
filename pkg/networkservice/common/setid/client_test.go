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

// Package setid_test provides a tests for package 'setid'
package setid_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setid"
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
	client := setid.NewClient("nse-3")
	conn, err := client.Request(context.Background(), request(connectionID, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_PathSegmentNameEqualClientName(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := setid.NewClient("nse-2")
	conn, err := client.Request(context.Background(), request(connectionID, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_PathSegmentIdEqualConnectionId(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := setid.NewClient("nse-3")
	conn, err := client.Request(context.Background(), request(pathSegmentID2, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, pathSegmentID2)
}

func TestNewClient_PathSegmentNameAndIDEqualClientNameAndID(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := setid.NewClient("nse-2")
	conn, err := client.Request(context.Background(), request(pathSegmentID2, 1))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, pathSegmentID2)
}

func TestNewClient_InvalidIndex(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := setid.NewClient("nse-3")
	conn, err := client.Request(context.Background(), request(connectionID, 2))
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, connectionID)
}

func TestNewClient_NoConnection(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := setid.NewClient("nse-3")
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}
