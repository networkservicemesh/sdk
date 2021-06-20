// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"context"
	"fmt"
	"time"

	"github.com/edwarnicke/grpcfd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

const (
	// RegistryExpiryDuration is a duration that should be used for expire tests
	RegistryExpiryDuration = time.Second
	// DialTimeout is a default dial timeout for the sandbox tests
	DialTimeout = 500 * time.Millisecond
)

type insecurePerRPCCredentials struct {
	credentials.PerRPCCredentials
}

func (i *insecurePerRPCCredentials) RequireTransportSecurity() bool {
	return false
}

// WithInsecureRPCCredentials makes passed call option with credentials.PerRPCCredentials insecure.
func WithInsecureRPCCredentials() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		for i := len(opts) - 1; i > -1; i-- {
			if v, ok := opts[i].(grpc.PerRPCCredsCallOption); ok {
				opts = append(opts, grpc.PerRPCCredentials(&insecurePerRPCCredentials{PerRPCCredentials: v.Creds}))
				break
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	})
}

// WithInsecureStreamRPCCredentials makes passed call option with credentials.PerRPCCredentials insecure.
func WithInsecureStreamRPCCredentials() grpc.DialOption {
	return grpc.WithChainStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		for i := len(opts) - 1; i > -1; i-- {
			if v, ok := opts[i].(grpc.PerRPCCredsCallOption); ok {
				opts = append(opts, grpc.PerRPCCredentials(&insecurePerRPCCredentials{PerRPCCredentials: v.Creds}))
				break
			}
		}
		return streamer(ctx, desc, cc, method, opts...)
	})
}

// GenerateTestToken generates test token
func GenerateTestToken(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Now().Add(time.Hour).Local(), nil
}

// GenerateExpiringToken returns a token generator with the specified expiration duration.
func GenerateExpiringToken(duration time.Duration) token.GeneratorFunc {
	value := fmt.Sprintf("TestToken-%s", duration)
	return func(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
		return value, time.Now().Add(duration).Local(), nil
	}
}

// DefaultDialOptions returns default dial options for sandbox testing
func DefaultDialOptions(genTokenFunc token.GeneratorFunc) []grpc.DialOption {
	return append([]grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(genTokenFunc)),
		),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
		WithInsecureRPCCredentials(),
		WithInsecureStreamRPCCredentials(),
	}, tracing.WithTracingDial()...)
}
