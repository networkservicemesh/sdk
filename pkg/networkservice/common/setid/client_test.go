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

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setid"
)

func TestNewClient_SetNewConnectionId(t *testing.T) {
	connectionID := "55ec76a7-d642-43ca-87bc-ab51765f575a"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connectionID,
			Path: &networkservice.Path{
				Index: 1,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nse-1",
						Id:   "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9",
					},
					{
						Name: "nse-2",
						Id:   "ece490ea-dfe8-4512-a3ca-5be7b39515c5",
					},
				},
			},
		},
	}
	client := setid.NewClient("nsc-2")
	conn, err := client.Request(context.Background(), request)
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.NotEqual(t, conn.Id, connectionID)
}

func TestNewClient_PathSegmentNameEqualClientName(t *testing.T) {
	connectionID := "54ec77a7-d645-43ca-87bc-ab51765f574a"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connectionID,
			Path: &networkservice.Path{
				Index: 1,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nse-1",
						Id:   "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9",
					},
					{
						Name: "nsc-2",
						Id:   "ece490ea-dfe8-4512-a3ca-5be7b39515c5",
					},
				},
			},
		},
	}
	client := setid.NewClient("nsc-2")
	conn, err := client.Request(context.Background(), request)
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, connectionID)
}

func TestNewClient_PathSegmentIdEqualConnectionId(t *testing.T) {
	connectionID := "54ed76a7-d642-43ca-87bc-ab51765f574a"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connectionID,
			Path: &networkservice.Path{
				Index: 1,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nse-1",
						Id:   "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9",
					},
					{
						Name: "nse-2",
						Id:   "54ed76a7-d642-43ca-87bc-ab51765f574a",
					},
				},
			},
		},
	}
	client := setid.NewClient("nsc-2")
	conn, err := client.Request(context.Background(), request)
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, connectionID)
}

func TestNewClient_InvalidIndex(t *testing.T) {
	connectionID := "54ec76a7-d642-44ca-87bc-ab51765g574a"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connectionID,
			Path: &networkservice.Path{
				Index: 2,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nse-1",
						Id:   "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9",
					},
					{
						Name: "nse-2",
						Id:   "ece490ea-dfe8-4512-a3ca-5be7b39515c5",
					},
				},
			},
		},
	}
	client := setid.NewClient("nsc-2")
	conn, err := client.Request(context.Background(), request)
	assert.NotNil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, conn.Id, connectionID)
}
