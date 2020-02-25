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

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setid"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func testEqual(id string) func(*testing.T, *networkservice.NetworkServiceRequest) {
	return func(t *testing.T, in *networkservice.NetworkServiceRequest) {
		assert.Equal(t, in.Connection.Id, id)
	}
}

func testNotEqual(id string) func(*testing.T, *networkservice.NetworkServiceRequest) {
	return func(t *testing.T, in *networkservice.NetworkServiceRequest) {
		assert.NotEqual(t, in.Connection.Id, id)
	}
}

var testData = []struct {
	name             string
	clientName       string
	path             *networkservice.Path
	connectionID     string
	testFuncProvider func(string) func(*testing.T, *networkservice.NetworkServiceRequest)
}{
	{"set new id for connection",
		"nsc-2",
		&networkservice.Path{
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
		"54ec76a7-d642-43ca-87bc-ab51765f574a",
		testNotEqual,
	},
	{"pathSegment.Name is the same as client name",
		"nsc-2",
		&networkservice.Path{
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
		"54ec76a7-d642-43ca-87bc-ab51765f574a",
		testEqual,
	},
	{"pathSegment.id is the same as request.Connection.Id",
		"nsc-2",
		&networkservice.Path{
			Index: 1,
			PathSegments: []*networkservice.PathSegment{
				{
					Name: "nse-1",
					Id:   "36ce7f0c-9f6d-40a4-8b39-6b56ff07eea9",
				},
				{
					Name: "nse-2",
					Id:   "54ec76a7-d642-43ca-87bc-ab51765f574a",
				},
			},
		},
		"54ec76a7-d642-43ca-87bc-ab51765f574a",
		testEqual,
	},
	{"invalid index",
		"nsc-2",
		&networkservice.Path{
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
		"54ec76a7-d642-43ca-87bc-ab51765f574a",
		testEqual,
	},
}

func Test_idClient_Request(t *testing.T) {
	for _, data := range testData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testClientRequest(t, test.clientName, test.path, test.connectionID, test.testFuncProvider(test.connectionID))
		})
	}
}

func testClientRequest(t *testing.T, clientName string, path *networkservice.Path, connectionID string, testFunc func(*testing.T, *networkservice.NetworkServiceRequest)) {
	client := next.NewNetworkServiceClient(setid.NewClient(clientName), checkrequest.NewClient(t, testFunc))
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:   connectionID,
			Path: path,
		},
	}
	_, _ = client.Request(context.Background(), request)
}
