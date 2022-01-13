// Copyright (c) 2020-2022 Cisco Systems, Inc.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package tracing provides a set of utilities to assist in working with opentelemetry
package tracing

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// WithTracing - returns array of grpc.ServerOption that should be passed to grpc.Dial to enable opentelemetry tracing
func WithTracing() []grpc.ServerOption {
	if opentelemetry.IsEnabled() {
		interceptor := func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp interface{}, err error) {
			return otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))(ctx, proto.Clone(req.(proto.Message)), info, handler)
		}
		return []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(
				interceptor),
			grpc.ChainStreamInterceptor(
				otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		}
	}
	return []grpc.ServerOption{
		grpc.EmptyServerOption{},
	}
}

// WithTracingDial returns array of grpc.DialOption that should be passed to grpc.Dial to enable opentelemetry tracing
func WithTracingDial() []grpc.DialOption {
	if opentelemetry.IsEnabled() {
		interceptor := func(
			ctx context.Context,
			method string,
			req, reply interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))(ctx, method, proto.Clone(req.(proto.Message)), reply, cc, invoker, opts...)
		}
		return []grpc.DialOption{
			grpc.WithChainUnaryInterceptor(
				interceptor),
			grpc.WithChainStreamInterceptor(
				otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		}
	}
	return []grpc.DialOption{
		grpc.EmptyDialOption{},
	}
}
