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

// Package mechanismtranslation provides client chain element to perform serverRequest -> clientRequest and
// clientConn -> serverConn mechanism translations
package mechanismtranslation

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type mechanismTranslationClient struct{}

// NewClient returns a new translation client chain element.
func NewClient() networkservice.NetworkServiceClient {
	return new(mechanismTranslationClient)
}

func (c *mechanismTranslationClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	// 1. Translate request mechanisms
	clientRequest := request.Clone()
	clientRequest.MechanismPreferences = nil

	clientRequest.Connection.Mechanism = load(ctx)

	// 2. Request client chain
	clientConn, err := next.Client(ctx).Request(ctx, clientRequest, opts...)
	if err != nil {
		return nil, err
	}
	store(ctx, clientConn.GetMechanism())

	// 3. Translate connection mechanism
	conn = clientConn.Clone()
	conn.Mechanism = request.GetConnection().GetMechanism()

	return conn, nil
}

func (c *mechanismTranslationClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// 1. Translate connection mechanism
	conn = conn.Clone()
	conn.Mechanism = loadAndDelete(ctx)

	// 2. Close client chain
	return next.Client(ctx).Close(ctx, conn, opts...)
}
