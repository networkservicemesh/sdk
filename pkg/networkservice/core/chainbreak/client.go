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

package chainbreak

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

// NewNetworkServiceClient wraps given chain element to execute Request, Close out of the chain
func NewNetworkServiceClient(wrapped networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(wrapped, &breakClient{})
}

type breakClient struct{}

func (t *breakClient) Request(_ context.Context, request *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	return request.GetConnection(), nil
}

func (t *breakClient) Close(_ context.Context, _ *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
