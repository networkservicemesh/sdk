// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco Systems, Inc.
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

// Package checkresponse - provides networkservice chain elements to check the response received from the next element in the chain
package checkresponse

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkResponseServer struct {
	*testing.T
	check func(*testing.T, *networkservice.Connection)
}

// NewServer - returns NetworkServiceServer chain elements to check the response received from the next element in the chain
//
//	t - *testing.T for checks
//	check - function to check the Connnection
func NewServer(t *testing.T, check func(*testing.T, *networkservice.Connection)) networkservice.NetworkServiceServer {
	return &checkResponseServer{
		T:     t,
		check: check,
	}
}

func (c *checkResponseServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	resp, err := next.Server(ctx).Request(ctx, request)
	c.check(c.T, resp)
	return resp, err
}

func (c *checkResponseServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
