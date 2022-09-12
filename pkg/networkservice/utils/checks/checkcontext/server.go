// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

package checkcontext

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkContextAfterServer struct {
	*testing.T
	check func(*testing.T, context.Context)
}

// NewServer - returns NetworkServiceServer that checks the context passed in from the previous Server in the chain
//
//	t - *testing.T used for the check
//	check - function that checks the context.Context
func NewServer(t *testing.T, check func(*testing.T, context.Context)) networkservice.NetworkServiceServer {
	return &checkContextAfterServer{
		T:     t,
		check: check,
	}
}

func (c *checkContextAfterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c.check(c.T, ctx)
	return next.Server(ctx).Request(ctx, request)
}

func (c *checkContextAfterServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.Server(ctx).Close(ctx, conn)
}
