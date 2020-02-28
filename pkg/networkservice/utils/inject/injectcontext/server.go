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

// Package injectcontext provides a NetworkServiceServer for injecting additional information into the context before
// calling the next chain element
package injectcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectContextServer struct {
	injector func(ctx context.Context) context.Context
}

// NewServer - returns a NetworkServiceServer chain element that calls injector to add additional information into
//             the context
func NewServer(injector func(ctx context.Context) context.Context) networkservice.NetworkServiceServer {
	return &injectContextServer{injector: injector}
}

func (i *injectContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = i.injector(ctx)
	return next.Server(ctx).Request(ctx, request)
}

func (i *injectContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	ctx = i.injector(ctx)
	return next.Server(ctx).Close(ctx, conn)
}
