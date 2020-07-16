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

package adapters

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
)

const testKey contextKeyType = "TestContext"

type writeServer struct {
}

func (w *writeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.Server(ctx).Request(ctx, request)
}

func (w *writeServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.Server(ctx).Close(ctx, connection)
}

func TestClientPassingContext(t *testing.T) {
	n := next.NewNetworkServiceClient(NewServerToClient(&writeServer{}), checkcontext.NewClient(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value(testKey).(bool); !ok {
			t.Error("Context not found")
		}
	}))

	_, err := n.Request(context.Background(), nil)
	require.NoError(t, err)

	_, err = n.Close(context.Background(), nil)
	require.NoError(t, err)
}
