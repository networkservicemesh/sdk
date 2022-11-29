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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectspiffeid"

	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type updateTokenNSEServerSuite struct {
	suite.Suite

	Token        string
	Expires      time.Time
	ProtoExpires *timestamppb.Timestamp
}

func (s *updateTokenNSEServerSuite) SetupSuite() {
	s.Token, s.Expires, _ = tokenGeneratorFunc()(nil)
	s.ProtoExpires = timestamppb.New(s.Expires)
}

func (s *updateTokenNSEServerSuite) Test_EmptyPathInRequest() {
	t := s.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()))

	ctx := grpcmetadata.PathWithContext(context.Background(), &grpcmetadata.Path{})

	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	// Note: Its up to authorization to decide that we won't accept requests without a Path from the client
	require.NoError(t, err)
	require.NotNil(t, nse)
}

func (s *updateTokenNSEServerSuite) Test_IndexInLastPositionAddNewSegment() {
	t := s.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
	nse := &registry.NetworkServiceEndpoint{}
	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()))

	path := &grpcmetadata.Path{
		Index: 1,
		PathSegments: []*networkservice.PathSegment{
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
	require.Equal(t, s.Token, path.PathSegments[2].Token)
	equalJSON(t, s.ProtoExpires, path.PathSegments[2].Expires)
}

func (s *updateTokenNSEServerSuite) Test_ValidIndexOverwriteValues() {
	t := s.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	path := &grpcmetadata.Path{
		Index: 1,
		PathSegments: []*networkservice.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
			{Name: "nsc-2", Id: "id-2"},
		},
	}

	expected := &grpcmetadata.Path{
		Index: 1,
		PathSegments: []*networkservice.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
			{Name: "nsc-2", Id: "id-2"},
		},
	}
	expected.PathSegments[2].Token = s.Token
	expected.PathSegments[2].Expires = s.ProtoExpires

	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)

	server := next.NewNetworkServiceEndpointRegistryServer(
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()))
	_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})

	require.NoError(t, err)
	equalJSON(t, expected, path)
}

func (s *updateTokenNSEServerSuite) Test_IndexGreaterThanArrayLength() {
	t := s.T()
	t.Cleanup(func() { goleak.VerifyNone(t) })
	path := &grpcmetadata.Path{
		Index: 2,
		PathSegments: []*networkservice.PathSegment{
			{Name: "nsc-0", Id: "id-0"},
			{Name: "nsc-1", Id: "id-1"},
		},
	}
	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)
	server := updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc())
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	assert.NotNil(t, err)
	assert.Nil(t, nse)
}

func (s *updateTokenNSEServerSuite) TestNSEChain() {
	t := s.T()
	path := &grpcmetadata.Path{
		Index:        0,
		PathSegments: []*networkservice.PathSegment{},
	}

	want := &grpcmetadata.Path{
		Index: 2,
		PathSegments: []*networkservice.PathSegment{
			{
				Name:    "nsc-1",
				Id:      "id-2",
				Token:   s.Token,
				Expires: s.ProtoExpires,
			}, {
				Name:    "local-nsm-1",
				Id:      "id-2",
				Token:   s.Token,
				Expires: s.ProtoExpires,
			}, {
				Name:    "remote-nsm-1",
				Id:      "id-2",
				Token:   s.Token,
				Expires: s.ProtoExpires,
			},
		},
	}

	elements := []registry.NetworkServiceEndpointRegistryServer{
		injectspiffeid.NewNetworkServiceEndpointRegistryServer("nsc-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()),
		injectspiffeid.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()),
		injectspiffeid.NewNetworkServiceEndpointRegistryServer("local-nsm-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		injectspiffeid.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()),
		injectspiffeid.NewNetworkServiceEndpointRegistryServer("remote-nsm-1"),
		updatepath.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc()),
	}

	ctx := context.Background()
	ctx = grpcmetadata.PathWithContext(ctx, path)

	server := next.NewNetworkServiceEndpointRegistryServer(elements...)
	_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{})
	require.Equal(t, 3, len(path.PathSegments))
	require.Equal(t, 0, int(path.Index))
	for i, seg := range path.PathSegments {
		require.Equal(t, want.PathSegments[i].Name, seg.Name)
		require.Equal(t, want.PathSegments[i].Token, seg.Token)
		equalJSON(t, want.PathSegments[i].Expires, s.ProtoExpires)
	}
	require.NoError(t, err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUpdateTokenNSEServerSuite(t *testing.T) {
	suite.Run(t, new(updateTokenNSEServerSuite))
}
