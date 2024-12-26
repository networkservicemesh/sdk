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
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
)

// WithTracing - returns array of grpc.ServerOption that should be passed to grpc.Dial to enable opentelemetry tracing
func WithTracing() []grpc.ServerOption {
	opts := []grpc.ServerOption{}
	if opentelemetry.IsEnabled() {
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	}
	return opts
}

// WithTracingDial returns array of grpc.DialOption that should be passed to grpc.Dial to enable opentelemetry tracing
func WithTracingDial() []grpc.DialOption {
	opts := []grpc.DialOption{}
	if opentelemetry.IsEnabled() {
		opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}
	return opts
}
