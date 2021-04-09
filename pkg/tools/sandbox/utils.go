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
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	// RegistryExpiryDuration is a duration that should be used for expire tests
	RegistryExpiryDuration = time.Second
	// DefaultTokenTimeout is a default token timeout for sandbox testing
	DefaultTokenTimeout = time.Hour
)

// GenerateTestToken generates test token
func GenerateTestToken(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Now().Add(DefaultTokenTimeout).Local(), nil
}

// GenerateExpiringToken returns a token generator with the specified expiration duration.
func GenerateExpiringToken(duration time.Duration) token.GeneratorFunc {
	value := fmt.Sprintf("TestToken-%s", duration)
	return func(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
		return value, time.Now().Add(duration).Local(), nil
	}
}

// NewCrossConnectClientFactory is a client.NewCrossConnectClientFactory with some fields preset for testing
func NewCrossConnectClientFactory(additionalFunctionality ...networkservice.NetworkServiceClient) client.Factory {
	return client.NewCrossConnectClientFactory(
		client.WithName(fmt.Sprintf("nsc-%v", uuid.New().String())),
		client.WithAdditionalFunctionality(additionalFunctionality...),
	)
}

// DefaultDialOptions returns default dial options for sandbox testing
func DefaultDialOptions(tokenTimeout time.Duration) []grpc.DialOption {
	return DefaultSecureDialOptions(newInsecureTC(), GenerateExpiringToken(tokenTimeout))
}

// DefaultSecureDialOptions returns default secure dial options for sandbox testing
func DefaultSecureDialOptions(clientTC credentials.TransportCredentials, genTokenFunc token.GeneratorFunc) []grpc.DialOption {
	return append([]grpc.DialOption{
		grpc.WithTransportCredentials(grpcfdTransportCredentials(clientTC)),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(genTokenFunc)),
		),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
	}, opentracing.WithTracingDial()...)
}

// Name creates unique name with the given prefix
func Name(prefix string) string {
	return prefix + "-" + uuid.New().String()
}

// SetupDefaultNode setups NSMgr and default Forwarder on the given node
func SetupDefaultNode(ctx context.Context, node *Node, supplyNSMgr SupplyNSMgrFunc) {
	node.NewNSMgr(ctx, Name("nsmgr"), nil, DefaultTokenTimeout, supplyNSMgr)

	node.NewForwarder(ctx, &registryapi.NetworkServiceEndpoint{
		Name: Name("forwarder"),
	}, DefaultTokenTimeout)
}
