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

// Package testerror provides networkservice chain elements that simply returns an error.  Useful for testing.
package testerror

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type testErrorClient struct{}

// NewClient - returns a testerror client that always returns an error when calling Request/Close
func NewClient() networkservice.NetworkServiceClient {
	return &testErrorClient{}
}

func (e *testErrorClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return nil, errors.New("error originates in testErrorClient")
}

func (e *testErrorClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, errors.New("error originates in testErrorClient")
}
