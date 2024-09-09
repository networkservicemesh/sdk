// Copyright (c) 2020-2022 Cisco Systems, Inc.
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

package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// tailClient is a simple implementation of networkservice.NetworkServiceClient that is called at the end of a chain
// to insure that we never call a method on a nil object.
type tailClient struct{}

func (t *tailClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	return request.GetConnection(), nil
}

func (t *tailClient) Close(ctx context.Context, _ *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
