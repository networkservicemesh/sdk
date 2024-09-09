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

// Package setextracontext define a chain element to set some extra context values
package setextracontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type setLabelsImpl struct {
	values map[string]string
}

// NewServer - construct a set set extra chain element, and pass a set of values we could like to be added to connection object.
func NewServer(values map[string]string) networkservice.NetworkServiceServer {
	return &setLabelsImpl{
		values: values,
	}
}

func (s *setLabelsImpl) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{}
	}
	if request.GetConnection().GetContext() == nil {
		request.Connection.Context = &networkservice.ConnectionContext{}
	}
	if request.Connection.Context.ExtraContext == nil {
		request.Connection.Context.ExtraContext = map[string]string{}
	}
	for k, v := range s.values {
		request.Connection.Context.ExtraContext[k] = v
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *setLabelsImpl) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	// No need to do anything on close
	return next.Server(ctx).Close(ctx, connection)
}
