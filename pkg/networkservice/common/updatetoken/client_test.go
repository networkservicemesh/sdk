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

// Package updatetoken_test provides tests for updateTokenClient and updateTokenServer chain elements
package updatetoken_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type updatePathClientSuite struct {
	suite.Suite

	Token        string
	Expires      time.Time
	ExpiresProto *timestamp.Timestamp
}

func (f *updatePathClientSuite) SetupSuite() {
	f.Token, f.Expires, _ = TokenGenerator(nil)
	f.ExpiresProto, _ = ptypes.TimestampProto(f.Expires)
}

func (f *updatePathClientSuite) TestNewClient_EmptyPathInRequest() {
	t := f.T()
	defer goleak.VerifyNone(t)
	client := next.NewNetworkServiceClient(updatepath.NewClient("nsc-1"), updatetoken.NewClient(TokenGenerator))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	expected := &networkservice.Connection{
		Id: "conn-1",
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{
				{
					Name:    "nsc-1",
					Id:      "conn-1",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.NoError(t, err)
	equalJSON(t, expected, conn)
}

func (f *updatePathClientSuite) TestNewClient_ZeroIndexAddNewSegment() {
	t := f.T()
	defer goleak.VerifyNone(t)
	client := next.NewNetworkServiceClient(updatepath.NewClient("nsc-1"), updatetoken.NewClient(TokenGenerator))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: []*networkservice.PathSegment{},
			},
		},
	}
	expected := &networkservice.Connection{
		Id: "conn-1",
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{
				{
					Name:    "nsc-1",
					Id:      "conn-1",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.NoError(t, err)
	equalJSON(t, expected, conn)
}

func (f *updatePathClientSuite) TestNewClient_ValidIndexOverwriteValues() {
	t := f.T()
	defer goleak.VerifyNone(t)
	client := next.NewNetworkServiceClient(updatepath.NewClient("nsc-1"), updatetoken.NewClient(TokenGenerator))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
			Path: &networkservice.Path{
				Index: 0,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nsc-0",
						Id:   "conn-0",
					}, {
						Name: "nsc-1",
						Id:   "conn-1",
					}, {
						Name: "nsc-2",
						Id:   "conn-2",
					},
				},
			},
		},
	}
	expected := &networkservice.Connection{
		Id: "conn-1",
		Path: &networkservice.Path{
			Index: 1,
			PathSegments: []*networkservice.PathSegment{
				{
					Name: "nsc-0",
					Id:   "conn-0",
				}, {
					Name:    "nsc-1",
					Id:      "conn-1",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				}, {
					Name: "nsc-2",
					Id:   "conn-2",
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.NoError(t, err)
	equalJSON(t, expected, conn)
}

func equalJSON(t require.TestingT, expected, actual interface{}) {
	json1, err1 := json.MarshalIndent(expected, "", "\t")
	require.NoError(t, err1)

	json2, err2 := json.MarshalIndent(actual, "", "\t")
	require.NoError(t, err2)
	require.Equal(t, string(json1), string(json2))
}

func TestNewClient_IndexGreaterThanArrayLength(t *testing.T) {
	defer goleak.VerifyNone(t)
	client := next.NewNetworkServiceClient(updatepath.NewClient("nsc-1"), updatetoken.NewClient(TokenGenerator))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
			Path: &networkservice.Path{
				Index: 2,
				PathSegments: []*networkservice.PathSegment{
					{
						Name: "nsc-0",
						Id:   "conn-0",
					}, {
						Name: "nsc-1",
						Id:   "conn-1",
					},
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUpdateTokenTestSuite(t *testing.T) {
	suite.Run(t, new(updatePathClientSuite))
}
