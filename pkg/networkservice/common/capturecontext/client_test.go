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

package capturecontext_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/capturecontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type contextKeyType string

const writerKey contextKeyType = "writerKey"

type checkContextClient struct{}

func (c *checkContextClient) Request(ctx context.Context, _ *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	return nil, checkContext(ctx)
}

func (c *checkContextClient) Close(ctx context.Context, _ *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	return nil, checkContext(ctx)
}

func checkContext(ctx context.Context) error {
	if capturecontext.CapturedContext(ctx).Value(writerKey) == true {
		return nil
	}
	return errors.New("Context not found")
}

type writeClient struct{}

func (w *writeClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return next.Client(ctx).Request(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func (w *writeClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func TestClientContextStorage(t *testing.T) {
	chain := next.NewNetworkServiceClient(&writeClient{}, capturecontext.NewClient(), &checkContextClient{})

	ctx := context.WithValue(context.Background(), writerKey, true)
	_, err := chain.Request(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Close(ctx, nil)
	require.NoError(t, err)
}
