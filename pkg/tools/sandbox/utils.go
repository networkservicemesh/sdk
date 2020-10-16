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

package sandbox

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// GenerateTestToken generates test token
func GenerateTestToken(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

// NewEndpoint creates endpoint and registers it into passed NSMgr.
func NewEndpoint(ctx context.Context, nse *registry.NetworkServiceEndpoint, generatorFunc token.GeneratorFunc, mgr nsmgr.Nsmgr, additionalFunctionality ...networkservice.NetworkServiceServer) (*EndpointEntry, error) {
	ep := endpoint.NewServer(ctx, nse.Name, authorize.NewServer(), generatorFunc, additionalFunctionality...)
	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	var err error
	if nse.Url != "" {
		u, err = url.Parse(nse.Url)
		if err != nil {
			return nil, err
		}
	}
	serve(ctx, u, ep.Register)
	if nse.Url == "" {
		nse.Url = u.String()
	}
	if nse.ExpirationTime == nil {
		deadline := time.Now().Add(time.Hour)
		expirationTime, err := ptypes.TimestampProto(deadline)
		if err != nil {
			return nil, err
		}
		nse.ExpirationTime = expirationTime
	}
	if _, err := mgr.NetworkServiceEndpointRegistryServer().Register(ctx, nse); err != nil {
		return nil, err
	}
	for _, service := range nse.NetworkServiceNames {
		if _, err := mgr.NetworkServiceRegistryServer().Register(ctx, &registry.NetworkService{Name: service, Payload: "IP"}); err != nil {
			return nil, err
		}
	}
	log.Entry(ctx).Infof("Started listen endpoint %v on %v.", nse.Name, u.String())
	return &EndpointEntry{Endpoint: ep, URL: u}, nil
}

// NewClient creates new networkservice.NetworkServiceClient configured to connect to the passed URL.
func NewClient(ctx context.Context, generatorFunc token.GeneratorFunc, connectTo *url.URL, additionalFunctionality ...networkservice.NetworkServiceClient) (networkservice.NetworkServiceClient, error) {
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(connectTo),
		append(spanhelper.WithTracingDial(), grpc.WithBlock(), grpc.WithInsecure())...)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = cc.Close()
	}()
	return client.NewClient(ctx, fmt.Sprintf("nsc-%v", uuid.New().String()), nil, generatorFunc, cc, additionalFunctionality...), nil
}
