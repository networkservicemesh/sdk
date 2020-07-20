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
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

var Token string
var Expires time.Time
var ExpiresProto *timestamp.Timestamp

func init() {
	Token, Expires, _ = TokenGenerator(nil)
	ExpiresProto, _ = ptypes.TimestampProto(Expires)
}

func TestNewServer_EmptyPathInRequest(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	server := updatepath.NewServer("nsc-1", TokenGenerator)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	conn, err := server.Request(context.Background(), request)
	// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}

func TestNewServer_IndexInLastPositionAddNewSegment(t *testing.T) {
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
					Token:   Token,
					Expires: ExpiresProto,
				},
			},
		},
	}
	server := updatepath.NewServer("nsc-2", TokenGenerator)
	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)
}

func TestNewServer_ValidIndexOverwriteValues(t *testing.T) {
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
						Name: "nsc-will-be-overwritten",
						Id:   "conn-will-be-overwritten",
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
					Token:   Token,
					Expires: ExpiresProto,
				},
			},
		},
	}
	server := updatepath.NewServer("nsc-2", TokenGenerator)
	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)
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
	server := updatepath.NewServer("nsc-2", TokenGenerator)
	conn, err := server.Request(context.Background(), request)
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}
