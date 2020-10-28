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

package updatetoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

type updateTokenServerSuite struct {
	suite.Suite

	Token        string
	Expires      time.Time
	ExpiresProto *timestamp.Timestamp
}

func (f *updateTokenServerSuite) SetupSuite() {
	f.Token, f.Expires, _ = TokenGenerator(nil)
	f.ExpiresProto, _ = ptypes.TimestampProto(f.Expires)
}

func (f *updateTokenServerSuite) TestNewServer_EmptyPathInRequest() {
	t := f.T()
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := next.NewNetworkServiceServer(updatepath.NewServer("nsc-1"), updatetoken.NewServer(TokenGenerator))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	conn, err := server.Request(context.Background(), request)
	// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
	require.NoError(t, err)
	require.NotNil(t, conn)
}

func (f *updateTokenServerSuite) TestNewServer_IndexInLastPositionAddNewSegment() {
	t := f.T()
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-2",
			Path: &networkservice.Path{
				Index: 1,
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
	server := next.NewNetworkServiceServer(updatepath.NewServer("nsc-2"), updatetoken.NewServer(TokenGenerator))
	conn, err := server.Request(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 3, len(conn.Path.PathSegments))
	require.Equal(t, "nsc-2", conn.Path.PathSegments[2].Name)
	require.Equal(t, conn.Id, conn.Path.PathSegments[2].Id)
	require.Equal(t, f.Token, conn.Path.PathSegments[2].Token)
	equalJSON(t, f.ExpiresProto, conn.Path.PathSegments[2].Expires)
}

func (f *updateTokenServerSuite) TestNewServer_ValidIndexOverwriteValues() {
	t := f.T()
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-2",
			Path: &networkservice.Path{
				Index: 1,
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
		Id: "conn-2",
		Path: &networkservice.Path{
			Index: 2,
			PathSegments: []*networkservice.PathSegment{
				{
					Name: "nsc-0",
					Id:   "conn-0",
				}, {
					Name: "nsc-1",
					Id:   "conn-1",
				}, {
					Name:    "nsc-2",
					Id:      "conn-2",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				},
			},
		},
	}
	server := next.NewNetworkServiceServer(updatepath.NewServer("nsc-2"), updatetoken.NewServer(TokenGenerator))
	conn, err := server.Request(context.Background(), request)
	assert.NoError(t, err)
	equalJSON(t, expected, conn)
}

func TestNewServer_IndexGreaterThanArrayLength(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
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
	server := updatetoken.NewServer(TokenGenerator)
	conn, err := server.Request(context.Background(), request)
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func (f *updateTokenServerSuite) TestChain() {
	t := f.T()
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-2",
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: []*networkservice.PathSegment{},
			},
		},
	}
	want := &networkservice.Connection{
		Id: "conn-2",
		Path: &networkservice.Path{
			Index: 2,
			PathSegments: []*networkservice.PathSegment{
				{
					Name:    "nsc-1",
					Id:      "conn-2",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				}, {
					Name:    "local-nsm-1",
					Id:      "conn-2",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				}, {
					Name:    "remote-nsm-1",
					Id:      "conn-2",
					Token:   f.Token,
					Expires: f.ExpiresProto,
				},
			},
		},
	}
	elements := []networkservice.NetworkServiceServer{
		adapters.NewClientToServer(next.NewNetworkServiceClient(updatepath.NewClient("nsc-1"), updatetoken.NewClient(TokenGenerator))),
		updatepath.NewServer("local-nsm-1"),
		updatetoken.NewServer(TokenGenerator),
		adapters.NewClientToServer(next.NewNetworkServiceClient(updatepath.NewClient("local-nsm-1"), updatetoken.NewClient(TokenGenerator))),
		updatepath.NewServer("remote-nsm-1"),
		updatetoken.NewServer(TokenGenerator),
		adapters.NewClientToServer(next.NewNetworkServiceClient(updatepath.NewClient("remote-nsm-1"), updatetoken.NewClient(TokenGenerator)))}

	server := next.NewNetworkServiceServer(elements...)

	got, err := server.Request(context.Background(), request)
	require.Equal(t, 3, len(got.Path.PathSegments))
	require.Equal(t, 0, int(got.Path.Index))
	for i, s := range got.Path.PathSegments {
		require.Equal(t, want.Path.PathSegments[i].Name, s.Name)
		require.Equal(t, want.Path.PathSegments[i].Token, s.Token)
		equalJSON(t, want.Path.PathSegments[i].Expires, s.Expires)
	}
	require.NoError(t, err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUpdateTokenServerTestSuite(t *testing.T) {
	suite.Run(t, new(updateTokenServerSuite))
}
