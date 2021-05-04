// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package nslabelsmatching provides registry elements for label matching
package nslabelsmatching

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
)

type labelsMatchingClient struct {
	Src map[string]string
}

func (s *labelsMatchingClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
}

func (s *labelsMatchingClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	rv, err := next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	return &labelsMatchingFindClient{rv, *s}, nil
}

func (s *labelsMatchingClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}

// NewNetworkServiceRegistryClient creates new instance of NetworkServiceRegistryClient
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	srv := &labelsMatchingClient{
		Src: map[string]string{},
	}

	clientinfo.AddClientInfo(context.Background(), srv.Src)

	return srv
}

type labelsMatchingFindClient struct {
	registry.NetworkServiceRegistry_FindClient
	labelsMatchingClient
}

func (tc *labelsMatchingFindClient) Recv() (*registry.NetworkService, error) {
	ns, err := tc.NetworkServiceRegistry_FindClient.Recv()
	if err != nil {
		return nil, err
	}

	for _, match := range ns.GetMatches() {
		for _, dest := range match.GetRoutes() {
			for key, val := range dest.GetDestinationSelector() {
				t, err := template.New(fmt.Sprintf("template-%v", key)).Parse(val)
				if err != nil {
					continue
				}

				var b bytes.Buffer
				err = t.Execute(&b, tc.labelsMatchingClient)
				if err != nil {
					continue
				}

				dest.DestinationSelector[key] = b.String()
			}
		}
	}

	return ns, err
}
