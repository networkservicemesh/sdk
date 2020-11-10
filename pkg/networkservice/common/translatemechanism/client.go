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

// Package translatemechanism contains client chain-element, which replaces incoming request mechanism
// and clears MechanismPreferences
package translatemechanism

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type translateMechanismClient struct {
	mechanism *networkservice.Mechanism
}

// NewClient - creates new translateMechanismClient chain element
func NewClient() networkservice.NetworkServiceClient {
	return &translateMechanismClient{}
}

func (c *translateMechanismClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	clientRequest := request.Clone()
	clientRequest.MechanismPreferences = nil
	clientRequest.Connection.Mechanism = c.mechanism
	conn, err := next.Client(ctx).Request(ctx, clientRequest)
	if err != nil {
		return nil, err
	}
	c.mechanism = conn.GetMechanism()
	return conn, nil
}

func (c *translateMechanismClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn = conn.Clone()
	conn.Mechanism = c.mechanism
	e, err := next.Client(ctx).Close(ctx, conn)
	c.mechanism = nil
	return e, err
}
