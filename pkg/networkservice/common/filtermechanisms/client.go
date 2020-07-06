// Copyright (c) 2019-2020 VMware, Inc.
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

// TODO find a better shorter name for this package

// Package filtermechanisms filters out remote mechanisms if communicating to/from a unix file socket,
// filters out local mechanisms otherwise.
package filtermechanisms

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type filterMechanismsClient struct{}

// NewClient - filters out remote mechanisms if connecting to a server over a unix file socket, otherwise filters
// out local mechanisms
func NewClient() networkservice.NetworkServiceClient {
	return &filterMechanismsClient{}
}

func (f *filterMechanismsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	u := clienturl.ClientURL(ctx)
	if u.Scheme == clienturl.UnixURLScheme {
		var mechanisms []*networkservice.Mechanism
		for _, mechanism := range request.GetMechanismPreferences() {
			if mechanism.Cls == cls.LOCAL {
				mechanisms = append(mechanisms, mechanism)
			}
		}
		request.MechanismPreferences = mechanisms
		return next.Client(ctx).Request(ctx, request, opts...)
	}
	var mechanisms []*networkservice.Mechanism
	for _, mechanism := range request.GetMechanismPreferences() {
		if mechanism.Cls == cls.REMOTE {
			mechanisms = append(mechanisms, mechanism)
		}
	}
	request.MechanismPreferences = mechanisms
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (f *filterMechanismsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
