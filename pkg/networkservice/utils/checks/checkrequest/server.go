// Copyright (c) 2020 Cisco and/or its affiliates.
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

package checkrequest

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkRequestAfterServer struct {
	*testing.T
	check func(*testing.T, *networkservice.NetworkServiceRequest)
}

// NewServer - returns NetworkServiceServer chain elements to check the request received from the previous element in the chain
//             t - *testing.T for checks
//             check - function to check the NetworkServiceRequest
func NewServer(t *testing.T, check func(*testing.T, *networkservice.NetworkServiceRequest)) networkservice.NetworkServiceServer {
	return &checkRequestAfterServer{
		T:     t,
		check: check,
	}
}

func (c *checkRequestAfterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c.check(c.T, request)
	return next.Server(ctx).Request(ctx, request)
}

func (c *checkRequestAfterServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
