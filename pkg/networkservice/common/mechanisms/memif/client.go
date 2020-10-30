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

// Package memif provides the necessary mechanisms to request and inject a memif socket.
package memif

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type memifMechanismClient struct {
	socketFileName string
}

// NewClient - returns client that sets memif preferred mechanism
func NewClient(socketFileName string) networkservice.NetworkServiceClient {
	return &memifMechanismClient{
		socketFileName: socketFileName,
	}
}

func (c *memifMechanismClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Append the mechanism preference to the request
	preferredMechanism := &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: memif.MECHANISM,
		Parameters: map[string]string{
			memif.SocketFileURL: (&url.URL{Scheme: memif.SocketFileScheme, Path: c.socketFileName}).String(),
		},
	}
	request.MechanismPreferences = append(request.MechanismPreferences, preferredMechanism)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *memifMechanismClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
