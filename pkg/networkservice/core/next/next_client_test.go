// Copyright (c) 2020 Doc.ai, Inc.
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
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func Test_NewNetworkServiceClient_ShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		NewNetworkServiceClient(&testNetworkServiceClient{})
	})
}

type testNetworkServiceClient struct {
}

func (*testNetworkServiceClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	return nil, nil
}
func (*testNetworkServiceClient) Close(ctx context.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}
