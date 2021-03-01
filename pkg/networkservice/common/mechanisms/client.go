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

package mechanisms

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

type mechanismsClient struct {
	mechanisms map[string]networkservice.NetworkServiceClient
}

// NewClient - returns a new mechanisms networkservicemesh.NetworkServiceClient
func NewClient(mechanisms map[string]networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	result := &mechanismsClient{
		mechanisms: make(map[string]networkservice.NetworkServiceClient),
	}
	for m, c := range mechanisms {
		result.mechanisms[m] = chain.NewNetworkServiceClient(c)
	}

	return result
}

func (mc *mechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetMechanism() != nil {
		srv, ok := mc.mechanisms[request.GetConnection().GetMechanism().GetType()]
		if ok {
			return srv.Request(ctx, request, opts...)
		}
		return nil, errUnsupportedMech
	}
	var err = errCannotSupportMech
	for _, mechanism := range request.GetMechanismPreferences() {
		cm, ok := mc.mechanisms[mechanism.GetType()]
		if ok {
			req := request.Clone()
			var resp *networkservice.Connection
			resp, respErr := cm.Request(ctx, req, opts...)
			if respErr == nil {
				return resp, nil
			}
			err = errors.Wrap(err, respErr.Error())
		}
	}
	return nil, err
}

func (mc *mechanismsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c, ok := mc.mechanisms[conn.GetMechanism().GetType()]
	if ok {
		return c.Close(ctx, conn)
	}
	return nil, errCannotSupportMech
}
