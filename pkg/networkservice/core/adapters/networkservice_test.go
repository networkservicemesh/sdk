// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package adapters_test

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
)

type contextKeyType string

const (
	contextPassingKey contextKeyType = "context"
	executionOrderKey contextKeyType = "order"
)

type (
	appendServer string
	appendClient string
)

var (
	reqCtx = context.WithValue(context.Background(), contextPassingKey, "req")
	clsCtx = context.WithValue(context.Background(), contextPassingKey, "cls")
)

func (a appendServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	defer appendContext(&ctx, "+", string(a))()
	return next.Server(ctx).Request(ctx, request)
}

func (a appendServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	defer appendContext(&ctx, "-", string(a))()
	return next.Server(ctx).Close(ctx, connection)
}

func (a appendClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	defer appendContext(&ctx, "+", string(a))()
	return next.Client(ctx).Request(ctx, in, opts...)
}

func (a appendClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	defer appendContext(&ctx, "-", string(a))()
	return next.Client(ctx).Close(ctx, in, opts...)
}

func appendContext(ctxPtr *context.Context, prefix, str string) func() {
	ctx := *ctxPtr

	// Context passing
	*ctxPtr = context.WithValue(ctx, contextPassingKey, ctx.Value(contextPassingKey).(string)+prefix+str)

	// Execution order: before
	*ctx.Value(executionOrderKey).(*string) += " (" + str

	return func() {
		// Execution order: after
		*ctx.Value(executionOrderKey).(*string) += " " + str + ")"
	}
}

type (
	errorClient struct{}
	errorServer struct{}
)

var errAlways = errors.New("errorClient and errorServer always return this error")

func (e *errorClient) Request(_ context.Context, _ *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	return nil, errAlways
}

func (e *errorClient) Close(_ context.Context, _ *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	return nil, errAlways
}

func (e *errorServer) Request(_ context.Context, _ *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return nil, errAlways
}

func (e *errorServer) Close(_ context.Context, _ *networkservice.Connection) (*empty.Empty, error) {
	return nil, errAlways
}

func TestNetworkService_Simple(t *testing.T) {
	checkOrderClient(t,
		next.NewNetworkServiceClient(
			adapters.NewServerToClient(appendServer("foo")),
			checkcontext.NewClient(t, checker("req+foo", "cls-foo")),
		),
		"! (foo foo)",
	)

	checkOrderServer(t,
		next.NewNetworkServiceServer(
			adapters.NewClientToServer(appendClient("foo")),
			checkcontext.NewServer(t, checker("req+foo", "cls-foo")),
		),
		"! (foo foo)",
	)
}

func TestNetworkService_Nested(t *testing.T) {
	checkOrderClient(t,
		next.NewNetworkServiceClient(
			appendClient("a"),
			adapters.NewServerToClient(next.NewNetworkServiceServer(
				appendServer("b"),
				adapters.NewClientToServer(appendClient("c")),
				appendServer("d"),
			)),
			appendClient("e"),
			checkcontext.NewClient(t, checker(
				"req+a+b+c+d+e",
				"cls-a-b-c-d-e",
			)),
		),
		"! (a (b (c (d (e e) d) c) b) a)",
	)

	checkOrderServer(t,
		next.NewNetworkServiceServer(
			appendServer("a"),
			adapters.NewClientToServer(next.NewNetworkServiceClient(
				appendClient("b"),
				adapters.NewServerToClient(appendServer("c")),
				appendClient("d"),
			)),
			appendServer("e"),
			checkcontext.NewServer(t, checker(
				"req+a+b+c+d+e",
				"cls-a-b-c-d-e",
			)),
		),
		"! (a (b (c (d (e e) d) c) b) a)",
	)
}

func TestNetworkService_DeepNested(t *testing.T) {
	checkOrderClient(t,
		next.NewNetworkServiceClient(
			appendClient("1"),
			adapters.NewServerToClient(next.NewNetworkServiceServer(
				appendServer("2"),
				adapters.NewClientToServer(next.NewNetworkServiceClient(
					appendClient("3"),
					adapters.NewServerToClient(
						next.NewNetworkServiceServer(
							appendServer("4"),
							adapters.NewClientToServer(next.NewNetworkServiceClient(
								appendClient("5"),
								adapters.NewServerToClient(appendServer("6")),
								appendClient("7"),
							)),
							adapters.NewClientToServer(appendClient("8")),
						)),
					adapters.NewServerToClient(appendServer("9")),
				)),
				adapters.NewClientToServer(appendClient("10")),
			)),
			adapters.NewServerToClient(appendServer("11")),
			checkcontext.NewClient(t, checker(
				"req+1+2+3+4+5+6+7+8+9+10+11",
				"cls-1-2-3-4-5-6-7-8-9-10-11",
			)),
		),
		"! (1 (2 (3 (4 (5 (6 (7 (8 (9 (10 (11 11) 10) 9) 8) 7) 6) 5) 4) 3) 2) 1)")

	checkOrderServer(t,
		next.NewNetworkServiceServer(
			appendServer("1"),
			adapters.NewClientToServer(next.NewNetworkServiceClient(
				appendClient("2"),
				adapters.NewServerToClient(next.NewNetworkServiceServer(
					appendServer("3"),
					adapters.NewClientToServer(
						next.NewNetworkServiceClient(
							appendClient("4"),
							adapters.NewServerToClient(next.NewNetworkServiceServer(
								appendServer("5"),
								adapters.NewClientToServer(appendClient("6")),
								appendServer("7"),
							)),
							adapters.NewServerToClient(appendServer("8")),
						)),
					adapters.NewClientToServer(appendClient("9")),
				)),
				adapters.NewServerToClient(appendServer("10")),
			)),
			adapters.NewClientToServer(appendClient("11")),
			checkcontext.NewServer(t, checker(
				"req+1+2+3+4+5+6+7+8+9+10+11",
				"cls-1-2-3-4-5-6-7-8-9-10-11",
			)),
		),
		"! (1 (2 (3 (4 (5 (6 (7 (8 (9 (10 (11 11) 10) 9) 8) 7) 6) 5) 4) 3) 2) 1)")
}

func TestNetworkService_DeepNestedError(t *testing.T) {
	checkOrderClient(t,
		next.NewNetworkServiceClient(
			appendClient("1"),
			adapters.NewServerToClient(next.NewNetworkServiceServer(
				appendServer("2"),
				adapters.NewClientToServer(next.NewNetworkServiceClient(
					appendClient("3"),
					adapters.NewServerToClient(
						next.NewNetworkServiceServer(
							appendServer("4"),
							adapters.NewClientToServer(next.NewNetworkServiceClient(
								appendClient("5"),
								&errorClient{},
								adapters.NewServerToClient(appendServer("6")),
								appendClient("7"),
							)),
							adapters.NewClientToServer(appendClient("8")),
						)),
					adapters.NewServerToClient(appendServer("9")),
				)),
				adapters.NewClientToServer(appendClient("10")),
			)),
			adapters.NewServerToClient(appendServer("11")),
			checkcontext.NewClient(t, func(t *testing.T, ctx context.Context) {
				t.Errorf("Should not be reached")
			}),
		),
		"! (1 (2 (3 (4 (5 5) 4) 3) 2) 1)")

	checkOrderServer(t,
		next.NewNetworkServiceServer(
			appendServer("1"),
			adapters.NewClientToServer(next.NewNetworkServiceClient(
				appendClient("2"),
				adapters.NewServerToClient(next.NewNetworkServiceServer(
					appendServer("3"),
					adapters.NewClientToServer(
						next.NewNetworkServiceClient(
							appendClient("4"),
							adapters.NewServerToClient(next.NewNetworkServiceServer(
								appendServer("5"),
								&errorServer{},
								adapters.NewClientToServer(appendClient("6")),
								appendServer("7"),
							)),
							adapters.NewServerToClient(appendServer("8")),
						)),
					adapters.NewClientToServer(appendClient("9")),
				)),
				adapters.NewServerToClient(appendServer("10")),
			)),
			adapters.NewClientToServer(appendClient("11")),
			checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
				t.Errorf("Should not be reached")
			}),
		),
		"! (1 (2 (3 (4 (5 5) 4) 3) 2) 1)")
}

func checkOrderClient(t *testing.T, c networkservice.NetworkServiceClient, expectedOrder string) {
	outRequest := "!"
	_, _ = c.Request(context.WithValue(reqCtx, executionOrderKey, &outRequest), nil)
	require.Equal(t, expectedOrder, outRequest)

	outClose := "!"
	_, _ = c.Close(context.WithValue(clsCtx, executionOrderKey, &outClose), nil)
	require.Equal(t, expectedOrder, outClose)
}

func checkOrderServer(t *testing.T, s networkservice.NetworkServiceServer, expectedOrder string) {
	outRequest := "!"
	_, _ = s.Request(context.WithValue(reqCtx, executionOrderKey, &outRequest), nil)
	require.Equal(t, expectedOrder, outRequest)

	outClose := "!"
	_, _ = s.Close(context.WithValue(clsCtx, executionOrderKey, &outClose), nil)
	require.Equal(t, expectedOrder, outClose)
}

func checker(expectedReq, expectedCls string) func(t *testing.T, ctx context.Context) {
	return func(t *testing.T, ctx context.Context) {
		require.Contains(t, []string{expectedReq, expectedCls}, ctx.Value(contextPassingKey))
	}
}

func BenchmarkServerToClient_Request(b *testing.B) {
	a := adapters.NewServerToClient(null.NewServer())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Request(context.Background(), nil)
	}
}

func BenchmarkClientToServer_Request(b *testing.B) {
	a := adapters.NewClientToServer(null.NewClient())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Request(context.Background(), nil)
	}
}

func BenchmarkNested20Adapters_Request(b *testing.B) {
	a := null.NewServer()
	for i := 0; i < 10; i++ {
		a = adapters.NewClientToServer(adapters.NewServerToClient(a))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Request(context.Background(), nil)
	}
}
