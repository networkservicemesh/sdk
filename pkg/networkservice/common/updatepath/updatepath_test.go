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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestChain(t *testing.T) {
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
					Token:   Token,
					Expires: ExpiresProto,
				}, {
					Name:    "local-nsm-1",
					Id:      "conn-2",
					Token:   Token,
					Expires: ExpiresProto,
				}, {
					Name:    "remote-nsm-1",
					Id:      "conn-2",
					Token:   Token,
					Expires: ExpiresProto,
				},
			},
		},
	}
	elements := []networkservice.NetworkServiceServer{
		adapters.NewClientToServer(updatepath.NewClient("nsc-1", TokenGenerator)),
		updatepath.NewServer("local-nsm-1", TokenGenerator),
		adapters.NewClientToServer(updatepath.NewClient("local-nsm-1", TokenGenerator)),
		updatepath.NewServer("remote-nsm-1", TokenGenerator),
		adapters.NewClientToServer(updatepath.NewClient("remote-nsm-1", TokenGenerator))}

	server := next.NewNetworkServiceServer(elements...)

	got, err := server.Request(context.Background(), request)
	assert.Equal(t, want, got)
	assert.NoError(t, err)
}
