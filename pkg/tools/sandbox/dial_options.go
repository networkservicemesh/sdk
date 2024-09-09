// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"github.com/edwarnicke/grpcfd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

type dialOpts struct {
	tokenGenerator token.GeneratorFunc
}

// DialOption is an option pattern for DialOptions.
type DialOption func(o *dialOpts)

// WithTokenGenerator sets tokenGenerator for DialOptions.
func WithTokenGenerator(tokenGenerator token.GeneratorFunc) DialOption {
	return func(opts *dialOpts) {
		opts.tokenGenerator = tokenGenerator
	}
}

// DialOptions is a helper method for building []grpc.DialOption for testing.
func DialOptions(options ...DialOption) []grpc.DialOption {
	tokenResetCh := make(chan struct{})
	close(tokenResetCh)

	opts := &dialOpts{
		tokenGenerator: GenerateTestToken,
	}
	for _, o := range options {
		o(opts)
	}

	return append([]grpc.DialOption{
		grpc.WithTransportCredentials(
			grpcfdTransportCredentials(insecure.NewCredentials()),
		),
		grpc.WithDefaultCallOptions(
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(opts.tokenGenerator)),
		),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
		WithInsecureRPCCredentials(),
		WithInsecureStreamRPCCredentials(),
	}, tracing.WithTracingDial()...)
}
