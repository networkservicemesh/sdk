// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package opentracing provides a set of utilities to assist in working with opentracing
package opentracing

import (
	"context"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/tools/jaeger"
)

// WithTracing - returns array of grpc.ServerOption that should be passed to grpc.Dial to enable opentracing
func WithTracing() []grpc.ServerOption {
	if jaeger.IsOpentracingEnabled() {
		interceptor := func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp interface{}, err error) {
			return otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())(ctx, proto.Clone(req.(proto.Message)), info, handler)
		}
		return []grpc.ServerOption{
			grpc.UnaryInterceptor(
				interceptor),
			grpc.StreamInterceptor(
				otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())),
		}
	}
	return []grpc.ServerOption{
		grpc.EmptyServerOption{},
	}
}

// WithTracingDial returns array of grpc.DialOption that should be passed to grpc.Dial to enable opentracing
func WithTracingDial() []grpc.DialOption {
	if jaeger.IsOpentracingEnabled() {
		interceptor := func(
			ctx context.Context,
			method string,
			req, reply interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer())(ctx, method, proto.Clone(req.(proto.Message)), reply, cc, invoker, opts...)
		}
		return []grpc.DialOption{
			grpc.WithUnaryInterceptor(
				interceptor),
			grpc.WithStreamInterceptor(
				otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())),
		}
	}
	return []grpc.DialOption{
		grpc.EmptyDialOption{},
	}
}
