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

// Package chainbreak provides wrapper chains elements to execute Request, Close on element out of the chain
package chainbreak

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// NewNetworkServiceServer wraps given chain element to execute Request, Close out of the chain
func NewNetworkServiceServer(wrapped networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(wrapped, &breakServer{})
}

type breakServer struct{}

func (t *breakServer) Request(_ context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return request.GetConnection(), nil
}

func (t *breakServer) Close(_ context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
