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

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// GenerateTestToken generates test token
func GenerateTestToken(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

// NewCrossConnectClientFactory is a client.NewCrossConnectClientFactory with some fields preset for testing
func NewCrossConnectClientFactory(generatorFunc token.GeneratorFunc, additionalFunctionality ...networkservice.NetworkServiceClient) func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
	return client.NewCrossConnectClientFactory(
		fmt.Sprintf("nsc-%v", uuid.New().String()),
		nil,
		generatorFunc,
		additionalFunctionality...)
}
