// Copyright (c) 2021 Cisco and/or its affiliates.
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

package connect_test

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// ExampleForwarder - example of how to use the connect chain element in a forwarder.
func ExampleNewServer() {
	var dialOptions []grpc.DialOption
	var callOptions []grpc.CallOption
	var additonalClientFunctionality []networkservice.NetworkServiceClient
	var beforeConnectServer1, beforeConnectServer2 networkservice.NetworkServiceServer
	var afterConnectServer1, afterConnectServer2 networkservice.NetworkServiceServer
	var chainCtx context.Context
	var tokenGenerator token.GeneratorFunc
	var name string
	forwarder := endpoint.NewServer(
		chainCtx,
		tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAdditionalFunctionality(
			beforeConnectServer1,
			beforeConnectServer2,
			connect.NewServer(
				client.NewClient(
					chainCtx,
					client.WithAdditionalFunctionality(additonalClientFunctionality...),
					client.WithDialOptions(dialOptions...),
					client.WithoutRefresh(),
					client.WithName(name),
				),
				callOptions...,
			),
			afterConnectServer1,
			afterConnectServer2,
		),
	)
	if forwarder != nil {
	}
}
