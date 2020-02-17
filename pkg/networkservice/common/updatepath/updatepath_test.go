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

func Test_updatePath_chained(t *testing.T) {
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
					Name: "nsc-1",
					Id:   "conn-2",
				}, {
					Name: "local-nsm-1",
					Id:   "conn-2",
				}, {
					Name: "remote-nsm-1",
					Id:   "conn-2",
				},
			},
		},
	}

	nscClient := updatepath.NewClient("nsc-1")
	localNsmServer := updatepath.NewServer("local-nsm-1")
	localNsmClient := updatepath.NewClient("local-nsm-1")
	remoteNsmServer := updatepath.NewServer("remote-nsm-1")
	remoteNsmClient := updatepath.NewClient("remote-nsm-1")

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(nscClient),
		localNsmServer,
		adapters.NewClientToServer(localNsmClient),
		remoteNsmServer,
		adapters.NewClientToServer(remoteNsmClient))

	got, err := server.Request(context.Background(), request)
	assert.Equal(t, want, got)
	assert.Equal(t, nil, err)
}
