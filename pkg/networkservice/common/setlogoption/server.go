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

// Package setlogoption - allow to overide some of logging capabilities.
package setlogoption

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type setLogOptionServer struct {
	options map[string]string
}

// NewServer - construct a new set log option server to override some logging capabilities for context.
func NewServer(options map[string]string) networkservice.NetworkServiceServer {
	return &setLogOptionServer{
		options: options,
	}
}

func (s *setLogOptionServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = s.withFields(ctx)
	return next.Server(ctx).Request(ctx, request)
}

func (s *setLogOptionServer) withFields(ctx context.Context) context.Context {
	fields := make(map[string]interface{})
	for k, v := range s.options {
		fields[k] = v
	}
	if len(fields) > 0 {
		ctx = logger.WithFields(ctx, fields)
	}
	return ctx
}

func (s *setLogOptionServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	ctx = s.withFields(ctx)
	return next.Server(ctx).Close(ctx, connection)
}
