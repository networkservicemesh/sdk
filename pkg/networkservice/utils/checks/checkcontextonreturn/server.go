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

// Package checkcontextonreturn - provides a NetworkServiceClient chain element for checking the state of the context.Context
//
//	after the next element in the chain has returned
package checkcontextonreturn

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkContextOnReturnServer struct {
	*testing.T
	check func(t *testing.T, ctx context.Context)
}

// NewServer - returns a NetworkServiceServer chain element for checking the state of the context.Context
//
//	after the next element in the chain has returned
//	t - *testing.T for doing the checks
//	check - function for checking the context.Context
func NewServer(t *testing.T, check func(t *testing.T, ctx context.Context)) networkservice.NetworkServiceServer {
	return &checkContextOnReturnServer{
		T:     t,
		check: check,
	}
}

func (t *checkContextOnReturnServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	t.check(t.T, ctx)
	return conn, err
}

func (t *checkContextOnReturnServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	e, err := next.Server(ctx).Close(ctx, conn)
	t.check(t.T, ctx)
	return e, err
}
