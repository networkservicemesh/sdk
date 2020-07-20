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

// Package updatepath_test provides tests for updatePathClient and updatePathServer chain elements
package updatepath_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
)

func TestNewClient_EmptyPathInRequest(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := updatepath.NewClient("nsc-1", TokenGenerator)
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
					Token:   Token,
					Expires: ExpiresProto,
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)
}

func TestNewClient_ZeroIndexAddNewSegment(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := updatepath.NewClient("nsc-1", TokenGenerator)
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
					Token:   Token,
					Expires: ExpiresProto,
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)
}

func TestNewClient_ValidIndexOverwriteValues(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := updatepath.NewClient("nsc-1", TokenGenerator)
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
						Name: "nsc-name-will-be-overwritten",
						Id:   "conn-will-be-overwritten",
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
					Token:   Token,
					Expires: ExpiresProto,
				}, {
					Name: "nsc-2",
					Id:   "conn-2",
				},
			},
		},
	}
	conn, err := client.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.Equal(t, expected, conn)
}

func TestNewClient_IndexGreaterThanArrayLength(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	client := updatepath.NewClient("nsc-1", TokenGenerator)
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
