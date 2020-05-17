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

package setlabels_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setlabels"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type testBulkRegisterNSEServer struct {
}

func (t *testBulkRegisterNSEServer) Send(s *registry.NSERegistration) error {
	return nil
}

func (t *testBulkRegisterNSEServer) Recv() (*registry.NSERegistration, error) {
	panic("implement me")
}

func (t *testBulkRegisterNSEServer) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (t testBulkRegisterNSEServer) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (t *testBulkRegisterNSEServer) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (t *testBulkRegisterNSEServer) Context() context.Context {
	return context.Background()
}

func (t testBulkRegisterNSEServer) SendMsg(m interface{}) error {
	panic("implement me")
}

func (t *testBulkRegisterNSEServer) RecvMsg(m interface{}) error {
	panic("implement me")
}

type assertServer struct {
	*testing.T
}

func (a *assertServer) RegisterNSE(context.Context, *registry.NSERegistration) (*registry.NSERegistration, error) {
	panic("implement me")
}

func (a *assertServer) BulkRegisterNSE(s registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	sample := &registry.NSERegistration{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: "",
	}, NetworkService: &registry.NetworkService{Name: "ns-1"}}
	require.Nil(a, s.Send(sample))
	require.NotNil(a, sample.GetNetworkServiceEndpoint().GetLabels())
	require.Len(a, sample.GetNetworkServiceEndpoint().GetLabels(), 1)
	require.Equal(a, "ns-1", sample.GetNetworkServiceEndpoint().GetLabels()["networkservicename"])
	return nil
}

func (a *assertServer) RemoveNSE(context.Context, *registry.RemoveNSERequest) (*empty.Empty, error) {
	panic("implement me")
}

func TestSetLabelsBulkRegisterNSEServer_Send(t *testing.T) {
	s := next.NewNetworkServiceRegistryServer(setlabels.NewServer(), &assertServer{T: t})
	_ = s.BulkRegisterNSE(&testBulkRegisterNSEServer{})
}
