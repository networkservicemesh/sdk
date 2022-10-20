// Copyright (c) 2022 Cisco and/or its affiliates.
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

// import (
// 	"context"
// 	"encoding/json"
// 	"strconv"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/suite"

// 	"github.com/stretchr/testify/require"

// 	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
// 	"github.com/networkservicemesh/sdk/pkg/registry/common/updatetoken"
// 	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

// 	"github.com/golang/protobuf/ptypes/timestamp"
// 	"go.uber.org/goleak"
// 	"google.golang.org/grpc/credentials"
// 	"google.golang.org/protobuf/types/known/timestamppb"

// 	"github.com/networkservicemesh/api/pkg/api/registry"
// 	"github.com/stretchr/testify/assert"
// )

// const (
// 	index = 0
// )

// func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
// 	return "TestToken" + strconv.Itoa(index), time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
// }

// type updateTokenServerSuite struct {
// 	suite.Suite

// 	Token        string
// 	Expires      time.Time
// 	ExpiresProto *timestamp.Timestamp
// }

// func (f *updateTokenServerSuite) SetupSuite() {
// 	f.Token, f.Expires, _ = TokenGenerator(nil)
// 	f.ExpiresProto = timestamppb.New(f.Expires)
// }

// func (f *updateTokenServerSuite) TestNewServer_EmptyPathInRequest() {
// 	t := f.T()
// 	t.Cleanup(func() { goleak.VerifyNone(t) })
// 	server := next.NewNetworkServiceEndpointRegistryServer(
// 		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-1"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator))

// 	nse := &registry.NetworkServiceEndpoint{}
// 	nse, err := server.Register(context.Background(), nse)
// 	// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
// 	require.NoError(t, err)
// 	require.NotNil(t, nse)
// }

// func (f *updateTokenServerSuite) TestNewServer_IndexInLastPositionAddNewSegment() {
// 	t := f.T()
// 	t.Cleanup(func() { goleak.VerifyNone(t) })
// 	nse := &registry.NetworkServiceEndpoint{
// 		Path: &registry.Path{
// 			Index: 1,
// 			PathSegments: []*registry.PathSegment{
// 				{Name: "nsc-0", Id: "id-0"},
// 				{Name: "nsc-1", Id: "id-1"},
// 			},
// 		},
// 	}
// 	server := next.NewNetworkServiceEndpointRegistryServer(
// 		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-2"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator))

// 	nse, err := server.Register(context.Background(), nse)
// 	require.NoError(t, err)
// 	require.Equal(t, 3, len(nse.Path.PathSegments))
// 	require.Equal(t, "nsc-2", nse.Path.PathSegments[2].Name)
// 	require.Equal(t, f.Token, nse.Path.PathSegments[2].Token)
// 	equalJSON(t, f.ExpiresProto, nse.Path.PathSegments[2].Expires)
// }

// func (f *updateTokenServerSuite) TestNewServer_ValidIndexOverwriteValues() {
// 	t := f.T()
// 	t.Cleanup(func() { goleak.VerifyNone(t) })

// 	nse := &registry.NetworkServiceEndpoint{
// 		Path: &registry.Path{
// 			Index: 1,
// 			PathSegments: []*registry.PathSegment{
// 				{Name: "nsc-0", Id: "id-0"},
// 				{Name: "nsc-1", Id: "id-1"},
// 				{Name: "nsc-2", Id: "id-2"},
// 			},
// 		},
// 	}

// 	expected := nse.Path.Clone()
// 	expected.PathSegments[2].Token = f.Token
// 	expected.PathSegments[2].Expires = f.ExpiresProto

// 	server := next.NewNetworkServiceEndpointRegistryServer(
// 		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-2"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator))
// 	nse, err := server.Register(context.Background(), nse)
// 	assert.NoError(t, err)
// 	equalJSON(t, expected, nse.Path)
// }

// func TestNewServer_IndexGreaterThanArrayLength(t *testing.T) {
// 	t.Cleanup(func() { goleak.VerifyNone(t) })
// 	nse := &registry.NetworkServiceEndpoint{
// 		Path: &registry.Path{
// 			Index: 2,
// 			PathSegments: []*registry.PathSegment{
// 				{Name: "nsc-0", Id: "id-0"},
// 				{Name: "nsc-1", Id: "id-1"},
// 			},
// 		},
// 	}

// 	server := updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator)
// 	nse, err := server.Register(context.Background(), nse)
// 	assert.NotNil(t, err)
// 	assert.Nil(t, nse)
// }

// func (f *updateTokenServerSuite) TestChain() {
// 	t := f.T()
// 	nse := &registry.NetworkServiceEndpoint{
// 		Path: &registry.Path{
// 			Index:        0,
// 			PathSegments: []*registry.PathSegment{},
// 		},
// 	}

// 	want := &registry.NetworkServiceEndpoint{
// 		Path: &registry.Path{
// 			Index: 2,
// 			PathSegments: []*registry.PathSegment{
// 				{
// 					Name:    "nsc-1",
// 					Id:      "id-2",
// 					Token:   f.Token,
// 					Expires: f.ExpiresProto,
// 				}, {
// 					Name:    "local-nsm-1",
// 					Id:      "id-2",
// 					Token:   f.Token,
// 					Expires: f.ExpiresProto,
// 				}, {
// 					Name:    "remote-nsm-1",
// 					Id:      "id-2",
// 					Token:   f.Token,
// 					Expires: f.ExpiresProto,
// 				},
// 			},
// 		},
// 	}
// 	elements := []registry.NetworkServiceEndpointRegistryServer{
// 		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-1"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator),
// 		updatepath.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator),
// 		updatepath.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
// 		updatepath.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator),
// 		updatepath.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
// 		updatetoken.NewNetworkServiceEndpointRegistryServer(TokenGenerator),
// 	}

// 	server := next.NewNetworkServiceEndpointRegistryServer(elements...)

// 	got, err := server.Register(context.Background(), nse)
// 	require.Equal(t, 3, len(got.Path.PathSegments))
// 	require.Equal(t, 0, int(got.Path.Index))
// 	for i, s := range got.Path.PathSegments {
// 		require.Equal(t, want.Path.PathSegments[i].Name, s.Name)
// 		require.Equal(t, want.Path.PathSegments[i].Token, s.Token)
// 		equalJSON(t, want.Path.PathSegments[i].Expires, s.Expires)
// 	}
// 	require.NoError(t, err)
// }

// // In order for 'go test' to run this suite, we need to create
// // a normal test function and pass our suite to suite.Run
// func TestUpdateTokenServerTestSuite(t *testing.T) {
// 	suite.Run(t, new(updateTokenServerSuite))
// }

// func equalJSON(t require.TestingT, expected, actual interface{}) {
// 	json1, err1 := json.MarshalIndent(expected, "", "\t")
// 	require.NoError(t, err1)

// 	json2, err2 := json.MarshalIndent(actual, "", "\t")
// 	require.NoError(t, err2)
// 	require.Equal(t, string(json1), string(json2))
// }
