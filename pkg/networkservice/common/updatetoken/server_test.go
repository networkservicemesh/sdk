// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	f.ExpiresProto = timestamppb.New(f.Expires)
}

func (f *updateTokenServerSuite) TestNewServer_EmptyPathInRequest() {
	t := f.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
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
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
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
	require.Equal(t, 3, len(conn.GetPath().GetPathSegments()))
	require.Equal(t, "nsc-2", conn.GetPath().GetPathSegments()[2].GetName())
	require.Equal(t, f.Token, conn.GetPath().GetPathSegments()[2].GetToken())
	equalJSON(t, f.ExpiresProto, conn.GetPath().GetPathSegments()[2].GetExpires())
}

func (f *updateTokenServerSuite) TestNewServer_ValidIndexOverwriteValues() {
	t := f.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
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

	expected := request.GetConnection().Clone()
	expected.Path.PathSegments[2].Token = f.Token
	expected.Path.PathSegments[2].Expires = f.ExpiresProto

	server := next.NewNetworkServiceServer(updatepath.NewServer("nsc-2"), updatetoken.NewServer(TokenGenerator))
	conn, err := server.Request(context.Background(), request)
	assert.NoError(t, err)
	equalJSON(t, expected, conn)
}

func TestNewServer_IndexGreaterThanArrayLength(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
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
		adapters.NewClientToServer(updatepath.NewClient("nsc-1")),
		updatetoken.NewServer(TokenGenerator),
		updatepath.NewServer("local-nsm-1"),
		updatetoken.NewServer(TokenGenerator),
		adapters.NewClientToServer(updatepath.NewClient("local-nsm-1")),
		updatepath.NewServer("remote-nsm-1"),
		updatetoken.NewServer(TokenGenerator),
		adapters.NewClientToServer(updatepath.NewClient("remote-nsm-1")),
		updatetoken.NewServer(TokenGenerator),
	}

	server := next.NewNetworkServiceServer(elements...)

	got, err := server.Request(context.Background(), request)
	require.Equal(t, 3, len(got.GetPath().GetPathSegments()))
	require.Equal(t, 0, int(got.GetPath().GetIndex()))
	for i, s := range got.GetPath().GetPathSegments() {
		require.Equal(t, want.GetPath().GetPathSegments()[i].GetName(), s.GetName())
		require.Equal(t, want.GetPath().GetPathSegments()[i].GetToken(), s.GetToken())
		equalJSON(t, want.GetPath().GetPathSegments()[i].GetExpires(), s.GetExpires())
	}
	require.NoError(t, err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUpdateTokenServerTestSuite(t *testing.T) {
	suite.Run(t, new(updateTokenServerSuite))
}

func equalJSON(t require.TestingT, expected, actual interface{}) {
	json1, err1 := json.MarshalIndent(expected, "", "\t")
	require.NoError(t, err1)

	json2, err2 := json.MarshalIndent(actual, "", "\t")
	require.NoError(t, err2)
	require.Equal(t, string(json1), string(json2))
}
