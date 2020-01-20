// Copyright (c) 2020 Cisco Systems, Inc.
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

package security

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// InsecureEnv environment variable, if "true" NSM will work in insecure mode
	InsecureEnv = "INSECURE"
)

// WithSpireDial - use Spire identity for TLS
func WithSpireDial(ctx context.Context) grpc.DialOption {
	insecure := viper.GetBool(InsecureEnv)
	if insecure {
		return grpc.EmptyDialOption{}
	}
	provider, err := NewSpireProvider(SpireAgentUnixAddr)
	if err != nil {
		return grpc.WithContextDialer(
			func(i context.Context, s string) (conn net.Conn, e error) {
				return nil, errors.New("some spire error here")
			})
	}
	cfg, _ := provider.GetTLSConfig(ctx)
	return grpc.WithTransportCredentials(credentials.NewTLS(cfg))
}

// WithSpire - use Spire identity for TLS
func WithSpire(ctx context.Context) grpc.ServerOption {
	insecure := viper.GetBool(InsecureEnv)
	if insecure {
		return grpc.EmptyServerOption{}
	}
	// TODO - decide what to do about error here
	provider, _ := NewSpireProvider(SpireAgentUnixAddr)
	cfg, _ := provider.GetTLSConfig(ctx)
	return grpc.Creds(credentials.NewTLS(cfg))
}
