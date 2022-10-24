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

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

const (
	index    = 0
	key      = "supersecret"
	spiffeid = "spiffe://test.com/server"
)

func tokenGeneratorFunc(id string) token.GeneratorFunc {
	return func(peerAuthInfo credentials.AuthInfo) (string, time.Time, error) {
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": id}).SignedString([]byte(key))
		return tok, time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), err
	}
}

type updateTokenServerSuite struct {
	suite.Suite

	Token        string
	Expires      time.Time
	ExpiresProto *timestamp.Timestamp
}

func (f *updateTokenServerSuite) SetupSuite() {
	f.Token, f.Expires, _ = tokenGeneratorFunc(spiffeid)(nil)
	f.ExpiresProto = timestamppb.New(f.Expires)
}

func (f *updateTokenServerSuite) TestNewServer_EmptyPathInRequest() {
	t := f.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-1"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc("spiffe://test.com/server")))

	ctx := grpcmetadata.PathWithContext(context.Background(), &registry.Path{})

	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
	require.NoError(t, err)
	require.NotNil(t, nse)
}

func (f *updateTokenServerSuite) TestNewServer_IndexInLastPositionAddNewSegment() {
	t := f.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
	nse := &registry.NetworkServiceEndpoint{}
	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-2"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)))

	path := &registry.Path{
		Index: 1,
		PathSegments: []*registry.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
		},
	}

	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)
	_, err := server.Register(ctx, nse)
	require.NoError(t, err)
	require.Equal(t, 3, len(path.PathSegments))
	require.Equal(t, "nsc-2", path.PathSegments[2].Name)
	require.Equal(t, f.Token, path.PathSegments[2].Token)
	equalJSON(t, f.ExpiresProto, path.PathSegments[2].Expires)
}

func (f *updateTokenServerSuite) TestNewServer_ValidIndexOverwriteValues() {
	t := f.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	path := &registry.Path{
		Index: 1,
		PathSegments: []*registry.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
			{Name: "nsc-2", Id: "id-2"},
		},
	}

	expected := &registry.Path{
		Index: 1,
		PathSegments: []*registry.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
			{Name: "nsc-2", Id: "id-2"},
		},
	}
	expected.PathSegments[2].Token = f.Token
	expected.PathSegments[2].Expires = f.ExpiresProto

	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)

	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-2"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)))
	_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})

	require.NoError(t, err)
	equalJSON(t, expected, path)
}

func TestNewServer_IndexGreaterThanArrayLength(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	path := &registry.Path{
		Index: 2,
		PathSegments: []*registry.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
		},
	}
	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)
	server := updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid))
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	assert.NotNil(t, err)
	assert.Nil(t, nse)
}

func (f *updateTokenServerSuite) TestChain() {
	t := f.T()
	path := &registry.Path{
		Index:        0,
		PathSegments: []*registry.PathSegment{},
	}

	want := &registry.Path{
		Index: 2,
		PathSegments: []*registry.PathSegment{
			{
				Name:    "nsc-1",
				Id:      "id-2",
				Token:   f.Token,
				Expires: f.ExpiresProto,
			}, {
				Name:    "local-nsm-1",
				Id:      "id-2",
				Token:   f.Token,
				Expires: f.ExpiresProto,
			}, {
				Name:    "remote-nsm-1",
				Id:      "id-2",
				Token:   f.Token,
				Expires: f.ExpiresProto,
			},
		},
	}

	elements := []registry.NetworkServiceEndpointRegistryServer{
		updatepath.NewNetworkServiceEndpointRegistryServer("nsc-1"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)),
		updatepath.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)),
		updatepath.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)),
		updatepath.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(spiffeid)),
	}

	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)

	server := next.NewNetworkServiceEndpointRegistryServer(elements...)
	_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	require.Equal(t, 3, len(path.PathSegments))
	require.Equal(t, 0, int(path.Index))
	for i, s := range path.PathSegments {
		require.Equal(t, want.PathSegments[i].Name, s.Name)
		require.Equal(t, want.PathSegments[i].Token, s.Token)
		equalJSON(t, want.PathSegments[i].Expires, s.Expires)
	}
	require.NoError(t, err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
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
